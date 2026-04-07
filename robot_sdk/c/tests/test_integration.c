/**
 * Integration tests for the Robomesh C SDK.
 *
 * Requires a running roboserver (use docker-compose.dev.yml).
 * Default: TCP on localhost:5001, HTTP on localhost:8080.
 *
 * Build:
 *   cd robot_sdk/c && mkdir build && cd build
 *   cmake .. && make
 *   ./test_integration
 */

#include "robomesh.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <assert.h>
#include <unistd.h>
#include <time.h>

/* Simple HTTP POST helper (uses system curl for admin operations) */
#include <sys/wait.h>

static const char *HOST = "localhost";
static int TCP_PORT = 5001;
static int HTTP_PORT = 8080;

static int tests_passed = 0;
static int tests_failed = 0;
static long test_run_id = 0;

/* Build a unique UUID for this test run to avoid PostgreSQL collisions */
static void make_uuid(char *out, size_t out_size, const char *prefix) {
    snprintf(out, out_size, "%s-%ld", prefix, test_run_id);
}

#define TEST(name) \
    do { printf("  %-50s ", name); } while(0)

#define PASS() \
    do { printf("PASS\n"); tests_passed++; } while(0)

#define FAIL(msg, ...) \
    do { printf("FAIL: " msg "\n", ##__VA_ARGS__); tests_failed++; } while(0)

/* ── Admin helper (calls curl) ─────────────────────────────── */

static char admin_token[2048] = {0};

static int admin_login(void) {
    char cmd[1024];
    snprintf(cmd, sizeof(cmd),
        "curl -s -X POST http://%s:%d/auth/login "
        "-H 'Content-Type: application/json' "
        "-d '{\"username\":\"admin\",\"password\":\"password1\"}'",
        HOST, HTTP_PORT);

    FILE *fp = popen(cmd, "r");
    if (!fp) return -1;

    char response[4096];
    size_t n = fread(response, 1, sizeof(response) - 1, fp);
    response[n] = '\0';
    pclose(fp);

    /* Extract token (simple parsing) */
    char *tok = strstr(response, "\"token\":\"");
    if (!tok) return -1;
    tok += 9;
    char *end = strchr(tok, '"');
    if (!end) return -1;
    size_t len = end - tok;
    if (len >= sizeof(admin_token)) return -1;
    memcpy(admin_token, tok, len);
    admin_token[len] = '\0';
    return 0;
}

static int provision_robot(const char *uuid, const char *pub_hex, const char *device_type) {
    char cmd[2048];
    snprintf(cmd, sizeof(cmd),
        "curl -s -o /dev/null -w '%%{http_code}' -X POST http://%s:%d/provision "
        "-H 'Content-Type: application/json' "
        "-H 'Authorization: Bearer %s' "
        "-d '{\"uuid\":\"%s\",\"public_key\":\"%s\",\"device_type\":\"%s\"}'",
        HOST, HTTP_PORT, admin_token, uuid, pub_hex, device_type);

    FILE *fp = popen(cmd, "r");
    if (!fp) return -1;

    char code[16];
    size_t n = fread(code, 1, sizeof(code) - 1, fp);
    code[n] = '\0';
    pclose(fp);

    return (strncmp(code, "201", 3) == 0) ? 0 : -1;
}

/* ── Tests ─────────────────────────────────────────────────── */

static void test_keypair_generation(void) {
    TEST("Key generation");
    robomesh_keypair_t kp;
    if (robomesh_generate_keypair(&kp) != ROBOMESH_OK) {
        FAIL("generate_keypair failed");
        return;
    }

    char pub[65], priv[65];
    robomesh_public_key_hex(&kp, pub);
    robomesh_private_key_hex(&kp, priv);

    if (strlen(pub) != 64 || strlen(priv) != 64) {
        FAIL("Bad hex lengths: pub=%zu priv=%zu", strlen(pub), strlen(priv));
        return;
    }
    PASS();
}

static void test_keypair_load(void) {
    TEST("Key loading from hex");
    robomesh_keypair_t kp;
    robomesh_err_t err = robomesh_load_keypair(
        "c55608b70c4a9f3b43bd1d23e86aaf4c3b2f4b823f54dc34ac668e85363ef2e1", &kp);
    if (err != ROBOMESH_OK) {
        FAIL("load_keypair failed");
        return;
    }

    char pub[65];
    robomesh_public_key_hex(&kp, pub);
    if (strcmp(pub, "b702036ee61847fdabecc07ce7da7b432c39aba98d1114c1c6f6f3f586ba98aa") != 0) {
        FAIL("Public key mismatch: %s", pub);
        return;
    }
    PASS();
}

static void test_auth_seeded_robot(void) {
    TEST("AUTH with seeded robot (example-001)");
    robomesh_keypair_t kp;
    robomesh_load_keypair(
        "c55608b70c4a9f3b43bd1d23e86aaf4c3b2f4b823f54dc34ac668e85363ef2e1", &kp);

    robomesh_config_t cfg = {
        .uuid = "example-001",
        .host = HOST,
        .tcp_port = TCP_PORT,
        .keypair = &kp,
    };
    robomesh_client_t *c = robomesh_client_create(&cfg);
    if (!c) { FAIL("create failed"); return; }

    robomesh_err_t err = robomesh_authenticate(c);
    if (err != ROBOMESH_OK) {
        FAIL("auth failed: %s", robomesh_last_error(c));
        robomesh_client_destroy(c);
        return;
    }

    const char *jwt = robomesh_get_jwt(c);
    if (!jwt || strlen(jwt) < 10) {
        FAIL("No JWT received");
        robomesh_client_destroy(c);
        return;
    }

    robomesh_client_destroy(c);
    PASS();
}

static void test_auth_unknown_robot(void) {
    TEST("AUTH with unknown robot (should fail)");
    robomesh_keypair_t kp;
    robomesh_generate_keypair(&kp);

    robomesh_config_t cfg = {
        .uuid = "nonexistent-c-robot",
        .host = HOST,
        .tcp_port = TCP_PORT,
        .keypair = &kp,
    };
    robomesh_client_t *c = robomesh_client_create(&cfg);
    if (!c) { FAIL("create failed"); return; }

    robomesh_err_t err = robomesh_authenticate(c);
    if (err == ROBOMESH_OK) {
        FAIL("Auth should have failed for unknown robot");
        robomesh_client_destroy(c);
        return;
    }

    robomesh_client_destroy(c);
    PASS();
}

static void test_auth_wrong_key(void) {
    TEST("AUTH with wrong private key (should fail)");

    /* Generate a fresh key and provision it */
    robomesh_keypair_t good_kp, bad_kp;
    robomesh_generate_keypair(&good_kp);
    robomesh_generate_keypair(&bad_kp);

    char pub[65];
    robomesh_public_key_hex(&good_kp, pub);

    char uuid[64];
    make_uuid(uuid, sizeof(uuid), "test-c-wrongkey");
    if (provision_robot(uuid, pub, "test_robot") < 0) {
        FAIL("Failed to provision robot");
        return;
    }

    /* Try to auth with the wrong key */
    robomesh_config_t cfg = {
        .uuid = uuid,
        .host = HOST,
        .tcp_port = TCP_PORT,
        .keypair = &bad_kp,  /* Wrong key! */
    };
    robomesh_client_t *c = robomesh_client_create(&cfg);
    if (!c) { FAIL("create failed"); return; }

    robomesh_err_t err = robomesh_authenticate(c);
    if (err == ROBOMESH_OK) {
        FAIL("Auth should have failed with wrong key");
        robomesh_client_destroy(c);
        return;
    }

    robomesh_client_destroy(c);
    PASS();
}

static void test_auth_provisioned_robot(void) {
    TEST("AUTH with freshly provisioned robot");

    robomesh_keypair_t kp;
    robomesh_generate_keypair(&kp);
    char pub[65];
    robomesh_public_key_hex(&kp, pub);

    char uuid[64];
    make_uuid(uuid, sizeof(uuid), "test-c-provisioned");
    if (provision_robot(uuid, pub, "test_robot") < 0) {
        FAIL("Failed to provision robot");
        return;
    }

    robomesh_config_t cfg = {
        .uuid = uuid,
        .host = HOST,
        .tcp_port = TCP_PORT,
        .keypair = &kp,
    };
    robomesh_client_t *c = robomesh_client_create(&cfg);
    if (!c) { FAIL("create failed"); return; }

    robomesh_err_t err = robomesh_authenticate(c);
    if (err != ROBOMESH_OK) {
        FAIL("auth failed: %s", robomesh_last_error(c));
        robomesh_client_destroy(c);
        return;
    }

    robomesh_client_destroy(c);
    PASS();
}

static void test_heartbeat(void) {
    TEST("Heartbeat after AUTH");

    robomesh_keypair_t kp;
    robomesh_generate_keypair(&kp);
    char pub[65];
    robomesh_public_key_hex(&kp, pub);

    char uuid[64];
    make_uuid(uuid, sizeof(uuid), "test-c-heartbeat");
    provision_robot(uuid, pub, "test_robot");

    robomesh_config_t cfg = {
        .uuid = uuid,
        .host = HOST,
        .tcp_port = TCP_PORT,
        .keypair = &kp,
    };
    robomesh_client_t *c = robomesh_client_create(&cfg);
    if (!c) { FAIL("create failed"); return; }

    robomesh_err_t err = robomesh_authenticate(c);
    if (err != ROBOMESH_OK) {
        FAIL("auth failed: %s", robomesh_last_error(c));
        robomesh_client_destroy(c);
        return;
    }

    /* Send heartbeats */
    err = robomesh_send_heartbeat(c, NULL, 0);
    if (err != ROBOMESH_OK) {
        FAIL("heartbeat 1 failed: %s", robomesh_last_error(c));
        robomesh_client_destroy(c);
        return;
    }

    err = robomesh_send_heartbeat(c, "{\"battery\":85}", 120);
    if (err != ROBOMESH_OK) {
        FAIL("heartbeat 2 failed: %s", robomesh_last_error(c));
        robomesh_client_destroy(c);
        return;
    }

    robomesh_client_destroy(c);
    PASS();
}

static void test_send_message(void) {
    TEST("Send message in session mode");

    robomesh_keypair_t kp;
    robomesh_generate_keypair(&kp);
    char pub[65];
    robomesh_public_key_hex(&kp, pub);

    char uuid[64];
    make_uuid(uuid, sizeof(uuid), "test-c-message");
    provision_robot(uuid, pub, "test_robot");

    robomesh_config_t cfg = {
        .uuid = uuid,
        .host = HOST,
        .tcp_port = TCP_PORT,
        .keypair = &kp,
    };
    robomesh_client_t *c = robomesh_client_create(&cfg);
    if (!c) { FAIL("create failed"); return; }

    if (robomesh_authenticate(c) != ROBOMESH_OK) {
        FAIL("auth failed: %s", robomesh_last_error(c));
        robomesh_client_destroy(c);
        return;
    }

    /* Send a message */
    robomesh_err_t err = robomesh_send(c, "hello from C SDK");
    if (err != ROBOMESH_OK) {
        FAIL("send failed: %s", robomesh_last_error(c));
        robomesh_client_destroy(c);
        return;
    }

    /* Try to receive the echo (with timeout) */
    char buf[1024];
    err = robomesh_recv(c, buf, sizeof(buf), 2000);
    if (err == ROBOMESH_OK) {
        /* Handler echoes back - good */
    }
    /* Timeout is also acceptable if handler isn't running */

    robomesh_client_destroy(c);
    PASS();
}

/* ── Main ──────────────────────────────────────────────────── */

int main(int argc, char *argv[]) {
    test_run_id = (long)time(NULL);
    if (argc > 1) HOST = argv[1];
    if (argc > 2) TCP_PORT = atoi(argv[2]);
    if (argc > 3) HTTP_PORT = atoi(argv[3]);

    printf("Robomesh C SDK Integration Tests\n");
    printf("Server: %s (TCP:%d, HTTP:%d)\n\n", HOST, TCP_PORT, HTTP_PORT);

    /* Login as admin for provisioning */
    printf("Logging in as admin...\n");
    if (admin_login() != 0) {
        fprintf(stderr, "FATAL: Failed to login as admin. Is the server running?\n");
        return 1;
    }
    printf("Admin login OK\n\n");

    printf("Running tests:\n");
    test_keypair_generation();
    test_keypair_load();
    test_auth_seeded_robot();
    test_auth_unknown_robot();
    test_auth_wrong_key();
    test_auth_provisioned_robot();
    test_heartbeat();
    test_send_message();

    printf("\nResults: %d passed, %d failed\n", tests_passed, tests_failed);
    return tests_failed > 0 ? 1 : 0;
}
