/**
 * Robomesh Robot SDK - C client library for communicating with Roboserver.
 *
 * Handles Ed25519 authentication, heartbeat, and TCP messaging.
 * Depends on OpenSSL (libcrypto) for Ed25519 and pthreads for background
 * heartbeat/receive threads.
 */

#ifndef ROBOMESH_H
#define ROBOMESH_H

#include <stdint.h>
#include <stdbool.h>
#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif

/* ── Error codes ───────────────────────────────────────────── */

typedef enum {
    ROBOMESH_OK = 0,
    ROBOMESH_ERR_CONNECT,
    ROBOMESH_ERR_AUTH,
    ROBOMESH_ERR_HEARTBEAT,
    ROBOMESH_ERR_SEND,
    ROBOMESH_ERR_RECV,
    ROBOMESH_ERR_CRYPTO,
    ROBOMESH_ERR_TIMEOUT,
    ROBOMESH_ERR_INVALID_ARG,
    ROBOMESH_ERR_ALLOC,
} robomesh_err_t;

/* ── Key management ────────────────────────────────────────── */

/** Ed25519 keypair (32-byte private seed + 32-byte public key). */
typedef struct {
    uint8_t private_key[32];  /* Ed25519 seed (private) */
    uint8_t public_key[32];   /* Ed25519 public key */
} robomesh_keypair_t;

/**
 * Generate a new Ed25519 keypair.
 * On error, the keypair is zeroed.
 */
robomesh_err_t robomesh_generate_keypair(robomesh_keypair_t *kp);

/**
 * Load a keypair from a hex string.
 * On error, the keypair is zeroed.
 * @param private_key_hex  64-char hex string (32-byte seed)
 * @param kp               Output keypair (public key derived from private)
 */
robomesh_err_t robomesh_load_keypair(const char *private_key_hex, robomesh_keypair_t *kp);

/**
 * Get the public key as a hex string.
 * @param kp    Keypair
 * @param out   Buffer of at least 65 bytes (64 hex + null)
 */
void robomesh_public_key_hex(const robomesh_keypair_t *kp, char *out);

/**
 * Get the private key (seed) as a hex string.
 * @param kp    Keypair
 * @param out   Buffer of at least 65 bytes (64 hex + null)
 */
void robomesh_private_key_hex(const robomesh_keypair_t *kp, char *out);

/* ── Client ────────────────────────────────────────────────── */

/** Opaque client handle. */
typedef struct robomesh_client robomesh_client_t;

/** Client configuration. */
typedef struct {
    const char *uuid;             /* Must be non-empty */
    const char *host;
    int tcp_port;
    const char *device_type;      /* Required for REGISTER, optional for AUTH */
    robomesh_keypair_t *keypair;
} robomesh_config_t;

/**
 * Create a new client. Must be freed with robomesh_client_destroy().
 * Returns NULL if uuid is empty or required fields are missing.
 */
robomesh_client_t *robomesh_client_create(const robomesh_config_t *config);

/**
 * Destroy a client and free all resources.
 * Stops background threads and zeroes key material.
 */
void robomesh_client_destroy(robomesh_client_t *client);

/**
 * Open TCP connection to roboserver.
 */
robomesh_err_t robomesh_connect(robomesh_client_t *client);

/**
 * Close the TCP connection. Stops background threads.
 */
void robomesh_disconnect(robomesh_client_t *client);

/**
 * Disconnect and reconnect to the server.
 * Does not re-authenticate — call robomesh_authenticate() after.
 */
robomesh_err_t robomesh_reconnect(robomesh_client_t *client);

/**
 * Perform AUTH challenge-response handshake.
 * On success, the JWT is stored internally and can be retrieved
 * with robomesh_get_jwt().
 */
robomesh_err_t robomesh_authenticate(robomesh_client_t *client);

/**
 * Perform REGISTER flow for a new robot.
 * Blocks until admin approves/rejects or timeout_secs expires.
 * timeout_secs is capped at 300 (server-side pending TTL).
 * device_type must match [a-zA-Z0-9_-]{1,64}.
 */
robomesh_err_t robomesh_register(robomesh_client_t *client, int timeout_secs);

/**
 * Send PERSIST command (after REGISTER) to move to permanent storage.
 */
robomesh_err_t robomesh_persist(robomesh_client_t *client);

/**
 * Get the JWT received from authentication. Returns NULL if not authenticated.
 */
const char *robomesh_get_jwt(const robomesh_client_t *client);

/**
 * Check if the client is connected.
 */
bool robomesh_is_connected(const robomesh_client_t *client);

/* ── Heartbeat ─────────────────────────────────────────────── */

/**
 * Send a heartbeat on a separate TCP connection (5s timeout).
 * Thread-safe: can be called from any thread.
 * @param extra_data_json  Optional extra data as JSON string, or NULL.
 *                         Must be < ~4000 bytes.
 * @param ttl              Custom TTL in seconds, or 0 for server default
 */
robomesh_err_t robomesh_send_heartbeat(robomesh_client_t *client,
                                        const char *extra_data_json,
                                        int ttl);

/**
 * Start a background thread that sends heartbeats at the given interval.
 * @param interval_secs  Seconds between heartbeats (must be > 0)
 * @param ttl            TTL for each heartbeat, or 0 for server default
 */
robomesh_err_t robomesh_start_heartbeat(robomesh_client_t *client,
                                         int interval_secs, int ttl);

/**
 * Stop the background heartbeat thread.
 */
void robomesh_stop_heartbeat(robomesh_client_t *client);

/* ── Messaging ─────────────────────────────────────────────── */

/**
 * Send a message to the server (in session mode, forwarded to handler).
 * On send failure, the connection is marked disconnected.
 */
robomesh_err_t robomesh_send(robomesh_client_t *client, const char *message);

/**
 * Receive a line from the server (blocking).
 * On recv failure, the connection is marked disconnected.
 * @param buf       Buffer to store the received line
 * @param buf_size  Size of the buffer
 * @param timeout_ms  Timeout in milliseconds, 0 for no timeout
 */
robomesh_err_t robomesh_recv(robomesh_client_t *client, char *buf, size_t buf_size,
                              int timeout_ms);

/** Callback type for incoming messages. */
typedef void (*robomesh_message_cb)(const char *message, void *user_data);

/**
 * Start a background thread that receives messages and invokes the callback.
 * Do not call robomesh_recv() while this thread is running.
 */
robomesh_err_t robomesh_start_recv(robomesh_client_t *client,
                                    robomesh_message_cb callback,
                                    void *user_data);

/**
 * Stop the background receive thread.
 */
void robomesh_stop_recv(robomesh_client_t *client);

/* ── Utility ───────────────────────────────────────────────── */

/**
 * Get the last error message (human-readable).
 */
const char *robomesh_last_error(const robomesh_client_t *client);

#ifdef __cplusplus
}
#endif

#endif /* ROBOMESH_H */
