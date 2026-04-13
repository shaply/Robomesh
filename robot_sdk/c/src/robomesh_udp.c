/**
 * Robomesh Robot SDK - UDP client implementation.
 *
 * JSON packet-based protocol over UDP with challenge-response auth.
 */

#include "robomesh_udp.h"
#include "robomesh.h"

#include <stdio.h>
#include <stdlib.h>
#include <stdarg.h>
#include <string.h>
#include <unistd.h>
#include <errno.h>
#include <time.h>
#include <pthread.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <netdb.h>
#include <poll.h>

#include <openssl/evp.h>

#define UDP_MAX_PACKET 65535
#define JWT_MAX 2048
#define ERR_MAX 512

/* ── Internal structures ──────────────────────────────────── */

struct robomesh_udp_client {
    char uuid[256];
    char host[256];
    int udp_port;
    robomesh_keypair_t keypair;
    int sock;
    struct sockaddr_in server_addr;
    char jwt[JWT_MAX];
    int64_t heartbeat_seq;
    char last_error[ERR_MAX];

    pthread_mutex_t mutex;

    /* Background heartbeat thread */
    pthread_t hb_thread;
    volatile bool hb_running;
    bool hb_started;
    int hb_interval;
    int hb_ttl;
};

/* ── Hex utilities (shared with robomesh.c) ──────────────── */

static void udp_bytes_to_hex(const uint8_t *bytes, size_t len, char *hex) {
    for (size_t i = 0; i < len; i++)
        sprintf(hex + i * 2, "%02x", bytes[i]);
    hex[len * 2] = '\0';
}

static int udp_hex_to_bytes(const char *hex, uint8_t *bytes, size_t max_len) {
    size_t hex_len = strlen(hex);
    if (hex_len % 2 != 0 || hex_len / 2 > max_len) return -1;
    for (size_t i = 0; i < hex_len / 2; i++) {
        unsigned int val;
        if (sscanf(hex + i * 2, "%2x", &val) != 1) return -1;
        bytes[i] = (uint8_t)val;
    }
    return (int)(hex_len / 2);
}

/* ── Ed25519 signing ─────────────────────────────────────── */

static robomesh_err_t udp_ed25519_sign(const robomesh_keypair_t *kp,
                                        const uint8_t *msg, size_t msg_len,
                                        uint8_t *sig, size_t *sig_len) {
    EVP_PKEY *pkey = EVP_PKEY_new_raw_private_key(EVP_PKEY_ED25519, NULL,
                                                    kp->private_key, 32);
    if (!pkey) return ROBOMESH_ERR_CRYPTO;

    EVP_MD_CTX *ctx = EVP_MD_CTX_new();
    if (!ctx) { EVP_PKEY_free(pkey); return ROBOMESH_ERR_CRYPTO; }

    robomesh_err_t err = ROBOMESH_OK;
    if (EVP_DigestSignInit(ctx, NULL, NULL, NULL, pkey) != 1 ||
        EVP_DigestSign(ctx, sig, sig_len, msg, msg_len) != 1) {
        err = ROBOMESH_ERR_CRYPTO;
    }

    EVP_MD_CTX_free(ctx);
    EVP_PKEY_free(pkey);
    return err;
}

/* ── Error handling ──────────────────────────────────────── */

static void udp_set_error(robomesh_udp_client_t *c, const char *fmt, ...) {
    va_list args;
    va_start(args, fmt);
    vsnprintf(c->last_error, ERR_MAX, fmt, args);
    va_end(args);
}

/* ── JSON helpers ────────────────────────────────────────── */

/* Simple JSON string value extractor. Writes value into out (max out_size).
   Returns 0 on success, -1 if key not found. */
static int json_get_string(const char *json, const char *key, char *out, size_t out_size) {
    char pattern[128];
    snprintf(pattern, sizeof(pattern), "\"%s\":\"", key);
    const char *start = strstr(json, pattern);
    if (!start) {
        /* Try with space after colon */
        snprintf(pattern, sizeof(pattern), "\"%s\": \"", key);
        start = strstr(json, pattern);
        if (!start) return -1;
    }
    start = strchr(start, ':');
    if (!start) return -1;
    start++; /* skip colon */
    while (*start == ' ') start++;
    if (*start != '"') return -1;
    start++; /* skip opening quote */

    size_t i = 0;
    while (*start && *start != '"' && i < out_size - 1) {
        if (*start == '\\' && *(start + 1)) {
            start++; /* skip escape */
        }
        out[i++] = *start++;
    }
    out[i] = '\0';
    return 0;
}

/* ── UDP I/O ─────────────────────────────────────────────── */

static int udp_send_packet(robomesh_udp_client_t *c, const char *json) {
    ssize_t n = sendto(c->sock, json, strlen(json), 0,
                       (struct sockaddr *)&c->server_addr, sizeof(c->server_addr));
    return (n > 0) ? 0 : -1;
}

static int udp_recv_packet(robomesh_udp_client_t *c, char *buf, size_t buf_size, int timeout_ms) {
    if (timeout_ms > 0) {
        struct pollfd pfd = { .fd = c->sock, .events = POLLIN };
        int ret = poll(&pfd, 1, timeout_ms);
        if (ret == 0) return -2; /* timeout */
        if (ret < 0) return -1;
    }

    ssize_t n = recvfrom(c->sock, buf, buf_size - 1, 0, NULL, NULL);
    if (n <= 0) return -1;
    buf[n] = '\0';
    return (int)n;
}

/* ── Client lifecycle ────────────────────────────────────── */

robomesh_udp_client_t *robomesh_udp_create(const robomesh_udp_config_t *config) {
    if (!config || !config->uuid || config->uuid[0] == '\0' ||
        !config->host || !config->keypair)
        return NULL;

    robomesh_udp_client_t *c = calloc(1, sizeof(*c));
    if (!c) return NULL;

    strncpy(c->uuid, config->uuid, sizeof(c->uuid) - 1);
    strncpy(c->host, config->host, sizeof(c->host) - 1);
    c->udp_port = config->udp_port > 0 ? config->udp_port : 5001;
    memcpy(&c->keypair, config->keypair, sizeof(robomesh_keypair_t));
    c->sock = -1;
    c->heartbeat_seq = (int64_t)time(NULL);
    pthread_mutex_init(&c->mutex, NULL);
    return c;
}

void robomesh_udp_destroy(robomesh_udp_client_t *client) {
    if (!client) return;
    robomesh_udp_disconnect(client);
    pthread_mutex_destroy(&client->mutex);
    memset(&client->keypair, 0, sizeof(robomesh_keypair_t));
    free(client);
}

robomesh_err_t robomesh_udp_connect(robomesh_udp_client_t *client) {
    if (!client) return ROBOMESH_ERR_INVALID_ARG;

    /* Resolve host */
    struct addrinfo hints = {0}, *result;
    hints.ai_family = AF_INET;
    hints.ai_socktype = SOCK_DGRAM;

    char port_str[16];
    snprintf(port_str, sizeof(port_str), "%d", client->udp_port);

    if (getaddrinfo(client->host, port_str, &hints, &result) != 0) {
        udp_set_error(client, "Failed to resolve host %s", client->host);
        return ROBOMESH_ERR_CONNECT;
    }

    memcpy(&client->server_addr, result->ai_addr, sizeof(client->server_addr));
    freeaddrinfo(result);

    client->sock = socket(AF_INET, SOCK_DGRAM, 0);
    if (client->sock < 0) {
        udp_set_error(client, "Failed to create UDP socket");
        return ROBOMESH_ERR_CONNECT;
    }

    /* Set default receive timeout */
    struct timeval tv = { .tv_sec = 10, .tv_usec = 0 };
    setsockopt(client->sock, SOL_SOCKET, SO_RCVTIMEO, &tv, sizeof(tv));

    return ROBOMESH_OK;
}

void robomesh_udp_disconnect(robomesh_udp_client_t *client) {
    if (!client) return;

    client->hb_running = false;
    if (client->hb_started) {
        pthread_join(client->hb_thread, NULL);
        client->hb_started = false;
    }

    if (client->sock >= 0) {
        close(client->sock);
        client->sock = -1;
    }
}

const char *robomesh_udp_get_jwt(const robomesh_udp_client_t *client) {
    if (!client || client->jwt[0] == '\0') return NULL;
    return client->jwt;
}

const char *robomesh_udp_last_error(const robomesh_udp_client_t *client) {
    if (!client) return "NULL client";
    return client->last_error;
}

/* ── AUTH flow ───────────────────────────────────────────── */

robomesh_err_t robomesh_udp_authenticate(robomesh_udp_client_t *client) {
    if (!client) return ROBOMESH_ERR_INVALID_ARG;

    if (client->sock < 0) {
        robomesh_err_t err = robomesh_udp_connect(client);
        if (err != ROBOMESH_OK) return err;
    }

    char packet[UDP_MAX_PACKET];
    char buf[UDP_MAX_PACKET];

    /* Step 1: Request nonce */
    snprintf(packet, sizeof(packet),
             "{\"type\":\"auth\",\"uuid\":\"%s\"}", client->uuid);

    if (udp_send_packet(client, packet) < 0) {
        udp_set_error(client, "Failed to send auth step 1");
        return ROBOMESH_ERR_SEND;
    }

    int n = udp_recv_packet(client, buf, sizeof(buf), 10000);
    if (n < 0) {
        udp_set_error(client, "Failed to receive auth step 1 response");
        return (n == -2) ? ROBOMESH_ERR_TIMEOUT : ROBOMESH_ERR_RECV;
    }

    /* Check for error */
    char status[64];
    if (json_get_string(buf, "status", status, sizeof(status)) < 0) {
        udp_set_error(client, "Invalid auth response (no status)");
        return ROBOMESH_ERR_AUTH;
    }
    if (strcmp(status, "error") == 0) {
        char err_msg[256];
        json_get_string(buf, "error", err_msg, sizeof(err_msg));
        udp_set_error(client, "Auth step 1 error: %s", err_msg);
        return ROBOMESH_ERR_AUTH;
    }
    if (strcmp(status, "nonce") != 0) {
        udp_set_error(client, "Expected nonce status, got: %s", status);
        return ROBOMESH_ERR_AUTH;
    }

    char nonce_hex[512];
    if (json_get_string(buf, "nonce", nonce_hex, sizeof(nonce_hex)) < 0) {
        udp_set_error(client, "No nonce in response");
        return ROBOMESH_ERR_AUTH;
    }

    /* Step 2: Sign nonce and send back */
    uint8_t nonce_bytes[256];
    int nonce_len = udp_hex_to_bytes(nonce_hex, nonce_bytes, sizeof(nonce_bytes));
    if (nonce_len < 0) {
        udp_set_error(client, "Invalid nonce hex");
        return ROBOMESH_ERR_AUTH;
    }

    uint8_t sig[64];
    size_t sig_len = sizeof(sig);
    robomesh_err_t err = udp_ed25519_sign(&client->keypair, nonce_bytes, nonce_len,
                                           sig, &sig_len);
    if (err != ROBOMESH_OK) {
        udp_set_error(client, "Failed to sign nonce");
        return err;
    }

    char sig_hex[129];
    udp_bytes_to_hex(sig, sig_len, sig_hex);

    snprintf(packet, sizeof(packet),
             "{\"type\":\"auth\",\"uuid\":\"%s\",\"nonce\":\"%s\",\"signature\":\"%s\"}",
             client->uuid, nonce_hex, sig_hex);

    if (udp_send_packet(client, packet) < 0) {
        udp_set_error(client, "Failed to send auth step 2");
        return ROBOMESH_ERR_SEND;
    }

    n = udp_recv_packet(client, buf, sizeof(buf), 10000);
    if (n < 0) {
        udp_set_error(client, "Failed to receive auth step 2 response");
        return (n == -2) ? ROBOMESH_ERR_TIMEOUT : ROBOMESH_ERR_RECV;
    }

    if (json_get_string(buf, "status", status, sizeof(status)) < 0) {
        udp_set_error(client, "Invalid auth response (no status)");
        return ROBOMESH_ERR_AUTH;
    }
    if (strcmp(status, "error") == 0) {
        char err_msg[256];
        json_get_string(buf, "error", err_msg, sizeof(err_msg));
        udp_set_error(client, "Auth step 2 error: %s", err_msg);
        return ROBOMESH_ERR_AUTH;
    }
    if (strcmp(status, "ok") != 0) {
        udp_set_error(client, "Expected ok status, got: %s", status);
        return ROBOMESH_ERR_AUTH;
    }

    if (json_get_string(buf, "jwt", client->jwt, JWT_MAX) < 0) {
        udp_set_error(client, "No JWT in auth response");
        return ROBOMESH_ERR_AUTH;
    }

    return ROBOMESH_OK;
}

/* ── Heartbeat ───────────────────────────────────────────── */

robomesh_err_t robomesh_udp_send_heartbeat(robomesh_udp_client_t *client,
                                            const char *extra_data_json,
                                            int ttl) {
    if (!client || client->sock < 0) return ROBOMESH_ERR_INVALID_ARG;

    /* Atomically increment sequence number */
    pthread_mutex_lock(&client->mutex);
    client->heartbeat_seq++;
    int64_t seq = client->heartbeat_seq;
    pthread_mutex_unlock(&client->mutex);

    /* Build payload JSON (compact, no spaces) */
    char payload[4096];
    int off = snprintf(payload, sizeof(payload), "{\"seq\":%lld", (long long)seq);
    if (ttl > 0)
        off += snprintf(payload + off, sizeof(payload) - off, ",\"ttl\":%d", ttl);
    if (extra_data_json)
        off += snprintf(payload + off, sizeof(payload) - off, ",\"extra_data\":%s", extra_data_json);
    int tail = snprintf(payload + off, sizeof(payload) - off, "}");
    if (off + tail >= (int)sizeof(payload)) {
        udp_set_error(client, "Heartbeat payload too large");
        return ROBOMESH_ERR_INVALID_ARG;
    }

    /* Sign the raw JSON bytes */
    uint8_t sig[64];
    size_t sig_len = sizeof(sig);
    robomesh_err_t err = udp_ed25519_sign(&client->keypair,
                                           (const uint8_t *)payload, strlen(payload),
                                           sig, &sig_len);
    if (err != ROBOMESH_OK) {
        udp_set_error(client, "Failed to sign heartbeat");
        return err;
    }

    char sig_hex[129];
    udp_bytes_to_hex(sig, sig_len, sig_hex);

    /* Build UDP packet — payload is a raw JSON object (not string) */
    char packet[UDP_MAX_PACKET];
    snprintf(packet, sizeof(packet),
             "{\"type\":\"heartbeat\",\"uuid\":\"%s\",\"payload\":%s,\"signature\":\"%s\"}",
             client->uuid, payload, sig_hex);

    if (udp_send_packet(client, packet) < 0) {
        udp_set_error(client, "Failed to send heartbeat");
        return ROBOMESH_ERR_SEND;
    }

    /* Wait for response */
    char buf[UDP_MAX_PACKET];
    int n = udp_recv_packet(client, buf, sizeof(buf), 5000);
    if (n < 0) {
        udp_set_error(client, "Failed to receive heartbeat response");
        return (n == -2) ? ROBOMESH_ERR_TIMEOUT : ROBOMESH_ERR_RECV;
    }

    char status[64];
    if (json_get_string(buf, "status", status, sizeof(status)) == 0 &&
        strcmp(status, "error") == 0) {
        char err_msg[256];
        json_get_string(buf, "error", err_msg, sizeof(err_msg));
        udp_set_error(client, "Heartbeat failed: %s", err_msg);
        return ROBOMESH_ERR_HEARTBEAT;
    }

    return ROBOMESH_OK;
}

static void *udp_heartbeat_thread_func(void *arg) {
    robomesh_udp_client_t *c = (robomesh_udp_client_t *)arg;
    while (c->hb_running) {
        robomesh_udp_send_heartbeat(c, NULL, c->hb_ttl);
        for (int i = 0; i < c->hb_interval * 10 && c->hb_running; i++)
            usleep(100000);
    }
    return NULL;
}

robomesh_err_t robomesh_udp_start_heartbeat(robomesh_udp_client_t *client,
                                             int interval_secs, int ttl) {
    if (!client || interval_secs <= 0) return ROBOMESH_ERR_INVALID_ARG;
    if (client->hb_started) return ROBOMESH_OK;

    client->hb_interval = interval_secs;
    client->hb_ttl = ttl;
    client->hb_running = true;

    if (pthread_create(&client->hb_thread, NULL, udp_heartbeat_thread_func, client) != 0) {
        client->hb_running = false;
        udp_set_error(client, "Failed to create heartbeat thread");
        return ROBOMESH_ERR_ALLOC;
    }
    client->hb_started = true;
    return ROBOMESH_OK;
}

void robomesh_udp_stop_heartbeat(robomesh_udp_client_t *client) {
    if (!client || !client->hb_started) return;
    client->hb_running = false;
    pthread_join(client->hb_thread, NULL);
    client->hb_started = false;
}

/* ── Messaging ───────────────────────────────────────────── */

robomesh_err_t robomesh_udp_send(robomesh_udp_client_t *client, const char *message) {
    if (!client || !message || client->sock < 0) return ROBOMESH_ERR_INVALID_ARG;
    if (client->jwt[0] == '\0') {
        udp_set_error(client, "Not authenticated");
        return ROBOMESH_ERR_AUTH;
    }

    /* Escape message for JSON embedding */
    char escaped[UDP_MAX_PACKET];
    size_t ei = 0;
    for (size_t i = 0; message[i] && ei < sizeof(escaped) - 2; i++) {
        char ch = message[i];
        if (ch == '"' || ch == '\\') {
            escaped[ei++] = '\\';
        } else if (ch == '\n') {
            escaped[ei++] = '\\';
            ch = 'n';
        } else if (ch == '\r') {
            escaped[ei++] = '\\';
            ch = 'r';
        } else if (ch == '\t') {
            escaped[ei++] = '\\';
            ch = 't';
        }
        escaped[ei++] = ch;
    }
    escaped[ei] = '\0';

    char packet[UDP_MAX_PACKET];
    snprintf(packet, sizeof(packet),
             "{\"type\":\"message\",\"uuid\":\"%s\",\"jwt\":\"%s\",\"payload\":\"%s\"}",
             client->uuid, client->jwt, escaped);

    if (udp_send_packet(client, packet) < 0) {
        udp_set_error(client, "Failed to send message");
        return ROBOMESH_ERR_SEND;
    }

    /* Wait for response */
    char buf[UDP_MAX_PACKET];
    int n = udp_recv_packet(client, buf, sizeof(buf), 5000);
    if (n < 0) {
        udp_set_error(client, "Failed to receive message response");
        return (n == -2) ? ROBOMESH_ERR_TIMEOUT : ROBOMESH_ERR_RECV;
    }

    char status[64];
    if (json_get_string(buf, "status", status, sizeof(status)) == 0 &&
        strcmp(status, "error") == 0) {
        char err_msg[256];
        json_get_string(buf, "error", err_msg, sizeof(err_msg));
        udp_set_error(client, "Message failed: %s", err_msg);
        return ROBOMESH_ERR_SEND;
    }

    return ROBOMESH_OK;
}

robomesh_err_t robomesh_udp_recv(robomesh_udp_client_t *client,
                                  char *buf, size_t buf_size,
                                  int timeout_ms) {
    if (!client || !buf || client->sock < 0) return ROBOMESH_ERR_INVALID_ARG;

    int n = udp_recv_packet(client, buf, buf_size, timeout_ms);
    if (n == -2) return ROBOMESH_ERR_TIMEOUT;
    if (n < 0) {
        udp_set_error(client, "Failed to receive");
        return ROBOMESH_ERR_RECV;
    }
    return ROBOMESH_OK;
}
