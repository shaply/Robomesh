/**
 * Example test robot using the Robomesh C SDK.
 *
 * Demonstrates: generate keys → provision via admin API → authenticate → heartbeat.
 *
 * Usage:
 *   ./test_robot [host] [tcp_port]
 *   Default: localhost:5001
 */

#include "robomesh.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <signal.h>
#include <unistd.h>

static volatile int running = 1;

static void sigint_handler(int sig) {
    (void)sig;
    running = 0;
}

int main(int argc, char *argv[]) {
    const char *host = argc > 1 ? argv[1] : "localhost";
    int tcp_port = argc > 2 ? atoi(argv[2]) : 5001;

    /* 1. Generate keypair */
    robomesh_keypair_t kp;
    if (robomesh_generate_keypair(&kp) != ROBOMESH_OK) {
        fprintf(stderr, "Failed to generate keypair\n");
        return 1;
    }

    char pub_hex[65], priv_hex[65];
    robomesh_public_key_hex(&kp, pub_hex);
    robomesh_private_key_hex(&kp, priv_hex);
    printf("Public key:  %s\n", pub_hex);
    printf("Private key: %s\n", priv_hex);

    /* NOTE: In a real scenario, you'd provision this key via the admin API first.
       For this example, use the pre-seeded example-001 robot instead. */
    printf("\nUsing pre-seeded robot 'example-001' for demo.\n");
    printf("(In production, provision the key above via POST /provision first.)\n\n");

    /* Load the pre-seeded test robot's key */
    robomesh_keypair_t seeded_kp;
    robomesh_load_keypair("c55608b70c4a9f3b43bd1d23e86aaf4c3b2f4b823f54dc34ac668e85363ef2e1",
                           &seeded_kp);

    /* 2. Create client */
    robomesh_config_t config = {
        .uuid = "example-001",
        .host = host,
        .tcp_port = tcp_port,
        .keypair = &seeded_kp,
    };
    robomesh_client_t *client = robomesh_client_create(&config);
    if (!client) {
        fprintf(stderr, "Failed to create client\n");
        return 1;
    }

    /* 3. Authenticate */
    printf("Authenticating...\n");
    robomesh_err_t err = robomesh_authenticate(client);
    if (err != ROBOMESH_OK) {
        fprintf(stderr, "Auth failed: %s\n", robomesh_last_error(client));
        robomesh_client_destroy(client);
        return 1;
    }
    printf("Authenticated! JWT: %.20s...%s\n",
           robomesh_get_jwt(client),
           robomesh_get_jwt(client) + strlen(robomesh_get_jwt(client)) - 10);

    /* 4. Send heartbeats */
    signal(SIGINT, sigint_handler);
    printf("Sending heartbeats. Press Ctrl+C to stop.\n");

    while (running) {
        err = robomesh_send_heartbeat(client, NULL, 60);
        if (err != ROBOMESH_OK) {
            fprintf(stderr, "Heartbeat error: %s\n", robomesh_last_error(client));
            break;
        }
        printf("Heartbeat OK\n");
        sleep(25);
    }

    /* 5. Cleanup */
    printf("Shutting down...\n");
    robomesh_client_destroy(client);
    return 0;
}
