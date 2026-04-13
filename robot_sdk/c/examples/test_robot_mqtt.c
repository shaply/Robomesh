/**
 * Example test robot using the Robomesh C SDK over MQTT.
 *
 * Demonstrates: connect to MQTT broker → authenticate → heartbeat → messaging.
 * Uses the pre-seeded example-001 robot.
 * Requires libmosquitto.
 *
 * Usage:
 *   ./test_robot_mqtt [host] [mqtt_port]
 *   Default: localhost:1883
 */

#include "robomesh_mqtt.h"
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

static void on_message(const char *message, void *user_data) {
    (void)user_data;
    printf("Received from handler: %s\n", message);
}

int main(int argc, char *argv[]) {
    const char *host = argc > 1 ? argv[1] : "localhost";
    int mqtt_port = argc > 2 ? atoi(argv[2]) : 1883;

    /* Load the pre-seeded test robot's key */
    robomesh_keypair_t kp;
    robomesh_load_keypair("c55608b70c4a9f3b43bd1d23e86aaf4c3b2f4b823f54dc34ac668e85363ef2e1",
                           &kp);

    char pub_hex[65];
    robomesh_public_key_hex(&kp, pub_hex);
    printf("Robot: example-001\n");
    printf("Public key: %s\n", pub_hex);
    printf("Protocol: MQTT on %s:%d\n\n", host, mqtt_port);

    /* Create MQTT client */
    robomesh_mqtt_config_t config = {
        .uuid = "example-001",
        .host = host,
        .mqtt_port = mqtt_port,
        .keypair = &kp,
    };
    robomesh_mqtt_client_t *client = robomesh_mqtt_create(&config);
    if (!client) {
        fprintf(stderr, "Failed to create MQTT client\n");
        return 1;
    }

    /* Connect to broker */
    printf("Connecting to MQTT broker...\n");
    robomesh_err_t err = robomesh_mqtt_connect(client);
    if (err != ROBOMESH_OK) {
        fprintf(stderr, "MQTT connect failed: %s\n", robomesh_mqtt_last_error(client));
        robomesh_mqtt_destroy(client);
        return 1;
    }
    printf("Connected to MQTT broker\n");

    /* Register message callback */
    robomesh_mqtt_on_message(client, on_message, NULL);

    /* Authenticate */
    printf("Authenticating over MQTT...\n");
    err = robomesh_mqtt_authenticate(client, 10000);
    if (err != ROBOMESH_OK) {
        fprintf(stderr, "MQTT auth failed: %s\n", robomesh_mqtt_last_error(client));
        robomesh_mqtt_destroy(client);
        return 1;
    }
    const char *jwt = robomesh_mqtt_get_jwt(client);
    printf("Authenticated! JWT: %.20s...%s\n", jwt, jwt + strlen(jwt) - 10);

    /* Send a test message */
    printf("Sending test message...\n");
    err = robomesh_mqtt_send(client, "hello from MQTT test robot");
    if (err != ROBOMESH_OK) {
        fprintf(stderr, "Send failed: %s\n", robomesh_mqtt_last_error(client));
    } else {
        printf("Message sent OK\n");
    }

    /* Send heartbeats */
    signal(SIGINT, sigint_handler);
    printf("Sending heartbeats over MQTT. Press Ctrl+C to stop.\n");

    while (running) {
        err = robomesh_mqtt_send_heartbeat(client, NULL, 60);
        if (err != ROBOMESH_OK) {
            fprintf(stderr, "Heartbeat error: %s\n", robomesh_mqtt_last_error(client));
            break;
        }
        printf("MQTT heartbeat OK\n");
        sleep(25);
    }

    printf("Shutting down...\n");
    robomesh_mqtt_stop_heartbeat(client);
    robomesh_mqtt_destroy(client);
    return 0;
}
