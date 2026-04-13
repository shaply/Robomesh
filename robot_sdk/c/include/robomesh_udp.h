/**
 * Robomesh Robot SDK - UDP client for communicating with Roboserver.
 *
 * Uses JSON packets over UDP with two-step challenge-response auth.
 * Depends on OpenSSL (libcrypto) for Ed25519.
 */

#ifndef ROBOMESH_UDP_H
#define ROBOMESH_UDP_H

#include "robomesh.h"

#ifdef __cplusplus
extern "C" {
#endif

/* ── UDP Client ───────────────────────────────────────────── */

/** Opaque UDP client handle. */
typedef struct robomesh_udp_client robomesh_udp_client_t;

/** UDP client configuration. */
typedef struct {
    const char *uuid;
    const char *host;
    int udp_port;             /* Default: 5001 */
    robomesh_keypair_t *keypair;
} robomesh_udp_config_t;

/**
 * Create a new UDP client. Must be freed with robomesh_udp_destroy().
 */
robomesh_udp_client_t *robomesh_udp_create(const robomesh_udp_config_t *config);

/**
 * Destroy a UDP client and free all resources.
 */
void robomesh_udp_destroy(robomesh_udp_client_t *client);

/**
 * Open the UDP socket.
 */
robomesh_err_t robomesh_udp_connect(robomesh_udp_client_t *client);

/**
 * Close the UDP socket and stop background threads.
 */
void robomesh_udp_disconnect(robomesh_udp_client_t *client);

/**
 * Perform two-step challenge-response authentication over UDP.
 * On success, the JWT is stored internally.
 */
robomesh_err_t robomesh_udp_authenticate(robomesh_udp_client_t *client);

/**
 * Get the JWT received from authentication. Returns NULL if not authenticated.
 */
const char *robomesh_udp_get_jwt(const robomesh_udp_client_t *client);

/* ── Heartbeat ────────────────────────────────────────────── */

/**
 * Send a signed heartbeat over UDP.
 * @param extra_data_json  Optional extra data as JSON string, or NULL
 * @param ttl              Custom TTL in seconds, or 0 for server default
 */
robomesh_err_t robomesh_udp_send_heartbeat(robomesh_udp_client_t *client,
                                            const char *extra_data_json,
                                            int ttl);

/**
 * Start a background thread that sends heartbeats at the given interval.
 */
robomesh_err_t robomesh_udp_start_heartbeat(robomesh_udp_client_t *client,
                                             int interval_secs, int ttl);

/**
 * Stop the background heartbeat thread.
 */
void robomesh_udp_stop_heartbeat(robomesh_udp_client_t *client);

/* ── Messaging ────────────────────────────────────────────── */

/**
 * Send a JWT-authenticated message to the handler via UDP.
 */
robomesh_err_t robomesh_udp_send(robomesh_udp_client_t *client, const char *message);

/**
 * Receive a UDP packet from the server (blocking with timeout).
 * @param buf        Buffer to store the received JSON
 * @param buf_size   Size of the buffer
 * @param timeout_ms Timeout in milliseconds, 0 for no timeout
 */
robomesh_err_t robomesh_udp_recv(robomesh_udp_client_t *client,
                                  char *buf, size_t buf_size,
                                  int timeout_ms);

/**
 * Get the last error message.
 */
const char *robomesh_udp_last_error(const robomesh_udp_client_t *client);

#ifdef __cplusplus
}
#endif

#endif /* ROBOMESH_UDP_H */
