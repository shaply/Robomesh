/**
 * Robomesh Robot SDK - MQTT client for communicating with Roboserver.
 *
 * Uses topic-based protocol with two-step challenge-response auth.
 * Depends on libmosquitto for MQTT and OpenSSL (libcrypto) for Ed25519.
 */

#ifndef ROBOMESH_MQTT_H
#define ROBOMESH_MQTT_H

#include "robomesh.h"

#ifdef __cplusplus
extern "C" {
#endif

/* ── MQTT Client ──────────────────────────────────────────── */

/** Opaque MQTT client handle. */
typedef struct robomesh_mqtt_client robomesh_mqtt_client_t;

/** MQTT client configuration. */
typedef struct {
    const char *uuid;
    const char *host;
    int mqtt_port;            /* Default: 1883 */
    robomesh_keypair_t *keypair;
} robomesh_mqtt_config_t;

/** Callback for incoming messages from handler. */
typedef void (*robomesh_mqtt_message_cb)(const char *message, void *user_data);

/**
 * Create a new MQTT client. Must be freed with robomesh_mqtt_destroy().
 */
robomesh_mqtt_client_t *robomesh_mqtt_create(const robomesh_mqtt_config_t *config);

/**
 * Destroy an MQTT client and free all resources.
 */
void robomesh_mqtt_destroy(robomesh_mqtt_client_t *client);

/**
 * Connect to the MQTT broker and start the network loop.
 */
robomesh_err_t robomesh_mqtt_connect(robomesh_mqtt_client_t *client);

/**
 * Disconnect from the MQTT broker and stop background threads.
 */
void robomesh_mqtt_disconnect(robomesh_mqtt_client_t *client);

/**
 * Check if the client is connected to the broker.
 */
bool robomesh_mqtt_is_connected(const robomesh_mqtt_client_t *client);

/**
 * Perform two-step challenge-response authentication over MQTT.
 * On success, the JWT is stored internally.
 * @param timeout_ms  Timeout for each auth step in milliseconds
 */
robomesh_err_t robomesh_mqtt_authenticate(robomesh_mqtt_client_t *client, int timeout_ms);

/**
 * Get the JWT received from authentication. Returns NULL if not authenticated.
 */
const char *robomesh_mqtt_get_jwt(const robomesh_mqtt_client_t *client);

/* ── Heartbeat ────────────────────────────────────────────── */

/**
 * Send a signed heartbeat over MQTT.
 * @param extra_data_json  Optional extra data as JSON string, or NULL
 * @param ttl              Custom TTL in seconds, or 0 for server default
 */
robomesh_err_t robomesh_mqtt_send_heartbeat(robomesh_mqtt_client_t *client,
                                             const char *extra_data_json,
                                             int ttl);

/**
 * Start a background thread that sends heartbeats at the given interval.
 */
robomesh_err_t robomesh_mqtt_start_heartbeat(robomesh_mqtt_client_t *client,
                                              int interval_secs, int ttl);

/**
 * Stop the background heartbeat thread.
 */
void robomesh_mqtt_stop_heartbeat(robomesh_mqtt_client_t *client);

/* ── Messaging ────────────────────────────────────────────── */

/**
 * Send a message to the handler via MQTT.
 * Unlike UDP, MQTT messages don't require JWT — access is controlled
 * by the broker's ACL hook at the topic level.
 */
robomesh_err_t robomesh_mqtt_send(robomesh_mqtt_client_t *client, const char *message);

/**
 * Register a callback for incoming messages from the handler.
 * Messages arrive on robomesh/to_robot/{uuid}.
 */
void robomesh_mqtt_on_message(robomesh_mqtt_client_t *client,
                               robomesh_mqtt_message_cb callback,
                               void *user_data);

/**
 * Get the last error message.
 */
const char *robomesh_mqtt_last_error(const robomesh_mqtt_client_t *client);

#ifdef __cplusplus
}
#endif

#endif /* ROBOMESH_MQTT_H */
