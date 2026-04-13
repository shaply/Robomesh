/**
 * Example test robot using the Robomesh C SDK over UDP.
 *
 * Demonstrates: authenticate over UDP → heartbeat → messaging.
 * Uses the pre-seeded example-001 robot.
 *
 * Usage:
 *   ./test_robot_udp [host] [udp_port]
 *   Default: localhost:5001
 */

#include "robomesh_udp.h"
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
    int udp_port = argc > 2 ? atoi(argv[2]) : 5001;

    /* Load the pre-seeded test robot's key */
    robomesh_keypair_t kp;
    robomesh_load_keypair("c55608b70c4a9f3b43bd1d23e86aaf4c3b2f4b823f54dc34ac668e85363ef2e1",
                           &kp);

    char pub_hex[65];
    robomesh_public_key_hex(&kp, pub_hex);
    printf("Robot: example-001\n");
    printf("Public key: %s\n", pub_hex);
    printf("Protocol: UDP on %s:%d\n\n", host, udp_port);

    /* Create UDP client */
    robomesh_udp_config_t config = {
        .uuid = "example-001",
        .host = host,
        .udp_port = udp_port,
        .keypair = &kp,
    };
    robomesh_udp_client_t *client = robomesh_udp_create(&config);
    if (!client) {
        fprintf(stderr, "Failed to create UDP client\n");
        return 1;
    }

    /* Authenticate over UDP */
    printf("Authenticating over UDP...\n");
    robomesh_err_t err = robomesh_udp_authenticate(client);
    if (err != ROBOMESH_OK) {
        fprintf(stderr, "UDP auth failed: %s\n", robomesh_udp_last_error(client));
        robomesh_udp_destroy(client);
        return 1;
    }
    const char *jwt = robomesh_udp_get_jwt(client);
    printf("Authenticated! JWT: %.20s...%s\n", jwt, jwt + strlen(jwt) - 10);

    /* Send a test message */
    printf("Sending test message...\n");
    err = robomesh_udp_send(client, "hello from UDP test robot");
    if (err != ROBOMESH_OK) {
        fprintf(stderr, "Send failed: %s\n", robomesh_udp_last_error(client));
    } else {
        printf("Message sent OK\n");
    }

    /* Send heartbeats */
    signal(SIGINT, sigint_handler);
    printf("Sending heartbeats over UDP. Press Ctrl+C to stop.\n");

    while (running) {
        err = robomesh_udp_send_heartbeat(client, NULL, 60);
        if (err != ROBOMESH_OK) {
            fprintf(stderr, "Heartbeat error: %s\n", robomesh_udp_last_error(client));
            break;
        }
        printf("UDP heartbeat OK\n");
        sleep(25);
    }

    printf("Shutting down...\n");
    robomesh_udp_destroy(client);
    return 0;
}
