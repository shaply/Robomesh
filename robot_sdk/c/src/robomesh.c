/**
 * Robomesh Robot SDK - C implementation.
 *
 * Uses OpenSSL for Ed25519 crypto, POSIX sockets for TCP, and pthreads for
 * background heartbeat/receive threads.
 */

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
#include <openssl/rand.h>

#define MAX_LINE 65536
#define JWT_MAX 2048
#define ERR_MAX 512
#define READ_BUF_SIZE 4096
#define MAX_REGISTER_TIMEOUT 300
#define HEARTBEAT_CONNECT_TIMEOUT 5

/* ── Internal structures ──────────────────────────────────── */

struct robomesh_client {
    char uuid[256];
    char host[256];
    int tcp_port;
    char device_type[64];
    robomesh_keypair_t keypair;
    int sock;
    bool connected;
    char jwt[JWT_MAX];
    int64_t heartbeat_seq;
    char last_error[ERR_MAX];

    pthread_mutex_t mutex;

    /* Buffered read state for main socket */
    char rbuf[READ_BUF_SIZE];
    size_t rbuf_pos;
    size_t rbuf_len;

    /* Background heartbeat thread */
    pthread_t hb_thread;
    volatile bool hb_running;
    bool hb_started;
    int hb_interval;
    int hb_ttl;

    /* Background receive thread */
    pthread_t recv_thread_handle;
    volatile bool recv_running;
    bool recv_started;
    robomesh_message_cb recv_cb;
    void *recv_cb_data;
};

/* ── Hex utilities ────────────────────────────────────────── */

static void bytes_to_hex(const uint8_t *bytes, size_t len, char *hex) {
    for (size_t i = 0; i < len; i++) {
        sprintf(hex + i * 2, "%02x", bytes[i]);
    }
    hex[len * 2] = '\0';
}

static int hex_to_bytes(const char *hex, uint8_t *bytes, size_t max_len) {
    size_t hex_len = strlen(hex);
    if (hex_len % 2 != 0 || hex_len / 2 > max_len) return -1;
    for (size_t i = 0; i < hex_len / 2; i++) {
        unsigned int val;
        if (sscanf(hex + i * 2, "%2x", &val) != 1) return -1;
        bytes[i] = (uint8_t)val;
    }
    return (int)(hex_len / 2);
}

/* ── Validation ──────────────────────────────────────────── */

static bool is_valid_device_type(const char *dt) {
    size_t len = strlen(dt);
    if (len == 0 || len > 64) return false;
    for (size_t i = 0; i < len; i++) {
        char c = dt[i];
        if (!((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
              (c >= '0' && c <= '9') || c == '_' || c == '-'))
            return false;
    }
    return true;
}

/* ── Ed25519 operations ───────────────────────────────────── */

static robomesh_err_t ed25519_sign(const robomesh_keypair_t *kp,
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

/* ── Key management ───────────────────────────────────────── */

robomesh_err_t robomesh_generate_keypair(robomesh_keypair_t *kp) {
    if (!kp) return ROBOMESH_ERR_INVALID_ARG;

    memset(kp, 0, sizeof(*kp));

    EVP_PKEY *pkey = NULL;
    EVP_PKEY_CTX *ctx = EVP_PKEY_CTX_new_id(EVP_PKEY_ED25519, NULL);
    if (!ctx) return ROBOMESH_ERR_CRYPTO;

    robomesh_err_t err = ROBOMESH_OK;
    if (EVP_PKEY_keygen_init(ctx) != 1 || EVP_PKEY_keygen(ctx, &pkey) != 1) {
        err = ROBOMESH_ERR_CRYPTO;
        goto cleanup;
    }

    size_t priv_len = 32, pub_len = 32;
    if (EVP_PKEY_get_raw_private_key(pkey, kp->private_key, &priv_len) != 1 ||
        EVP_PKEY_get_raw_public_key(pkey, kp->public_key, &pub_len) != 1) {
        err = ROBOMESH_ERR_CRYPTO;
        memset(kp, 0, sizeof(*kp));
    }

cleanup:
    EVP_PKEY_free(pkey);
    EVP_PKEY_CTX_free(ctx);
    return err;
}

robomesh_err_t robomesh_load_keypair(const char *private_key_hex, robomesh_keypair_t *kp) {
    if (!private_key_hex || !kp) return ROBOMESH_ERR_INVALID_ARG;

    memset(kp, 0, sizeof(*kp));

    if (hex_to_bytes(private_key_hex, kp->private_key, 32) != 32) {
        memset(kp, 0, sizeof(*kp));
        return ROBOMESH_ERR_INVALID_ARG;
    }

    /* Derive public key from private seed */
    EVP_PKEY *pkey = EVP_PKEY_new_raw_private_key(EVP_PKEY_ED25519, NULL,
                                                    kp->private_key, 32);
    if (!pkey) {
        memset(kp, 0, sizeof(*kp));
        return ROBOMESH_ERR_CRYPTO;
    }

    size_t pub_len = 32;
    robomesh_err_t err = ROBOMESH_OK;
    if (EVP_PKEY_get_raw_public_key(pkey, kp->public_key, &pub_len) != 1) {
        err = ROBOMESH_ERR_CRYPTO;
        memset(kp, 0, sizeof(*kp));
    }

    EVP_PKEY_free(pkey);
    return err;
}

void robomesh_public_key_hex(const robomesh_keypair_t *kp, char *out) {
    bytes_to_hex(kp->public_key, 32, out);
}

void robomesh_private_key_hex(const robomesh_keypair_t *kp, char *out) {
    bytes_to_hex(kp->private_key, 32, out);
}

/* ── TCP helpers ──────────────────────────────────────────── */

static int tcp_connect_with_timeout(const char *host, int port, int timeout_secs) {
    struct addrinfo hints = {0}, *result, *rp;
    hints.ai_family = AF_UNSPEC;
    hints.ai_socktype = SOCK_STREAM;

    char port_str[16];
    snprintf(port_str, sizeof(port_str), "%d", port);

    if (getaddrinfo(host, port_str, &hints, &result) != 0)
        return -1;

    int sock = -1;
    for (rp = result; rp; rp = rp->ai_next) {
        sock = socket(rp->ai_family, rp->ai_socktype, rp->ai_protocol);
        if (sock == -1) continue;

        struct timeval tv = { .tv_sec = timeout_secs, .tv_usec = 0 };
        setsockopt(sock, SOL_SOCKET, SO_RCVTIMEO, &tv, sizeof(tv));
        setsockopt(sock, SOL_SOCKET, SO_SNDTIMEO, &tv, sizeof(tv));

        if (connect(sock, rp->ai_addr, rp->ai_addrlen) == 0) break;
        close(sock);
        sock = -1;
    }
    freeaddrinfo(result);
    return sock;
}

static int tcp_connect(const char *host, int port) {
    return tcp_connect_with_timeout(host, port, 30);
}

static int send_line(int sock, const char *line) {
    size_t len = strlen(line);
    char *buf = malloc(len + 2);
    if (!buf) return -1;
    memcpy(buf, line, len);
    buf[len] = '\n';
    buf[len + 1] = '\0';

    ssize_t total = 0;
    while ((size_t)total < len + 1) {
        ssize_t n = send(sock, buf + total, len + 1 - total, 0);
        if (n <= 0) { free(buf); return -1; }
        total += n;
    }
    free(buf);
    return 0;
}

/* Byte-at-a-time recv for temporary sockets (heartbeat responses) */
static int recv_line_raw(int sock, char *buf, size_t buf_size) {
    size_t pos = 0;
    while (pos < buf_size - 1) {
        char c;
        ssize_t n = recv(sock, &c, 1, 0);
        if (n <= 0) return -1;
        if (c == '\n') break;
        buf[pos++] = c;
    }
    buf[pos] = '\0';
    return (int)pos;
}

/* Buffered recv for the client's main socket */
static int recv_line_buffered(robomesh_client_t *c, char *buf, size_t buf_size) {
    size_t out_pos = 0;
    while (out_pos < buf_size - 1) {
        /* Drain existing buffer data */
        while (c->rbuf_pos < c->rbuf_len && out_pos < buf_size - 1) {
            char ch = c->rbuf[c->rbuf_pos++];
            if (ch == '\n') {
                buf[out_pos] = '\0';
                return (int)out_pos;
            }
            buf[out_pos++] = ch;
        }
        if (out_pos >= buf_size - 1) break;

        /* Refill buffer from socket */
        ssize_t n = recv(c->sock, c->rbuf, READ_BUF_SIZE, 0);
        if (n <= 0) return -1;
        c->rbuf_pos = 0;
        c->rbuf_len = (size_t)n;
    }
    buf[out_pos] = '\0';
    return (int)out_pos;
}

/* ── Client lifecycle ─────────────────────────────────────── */

static void set_error(robomesh_client_t *c, const char *fmt, ...) {
    va_list args;
    va_start(args, fmt);
    vsnprintf(c->last_error, ERR_MAX, fmt, args);
    va_end(args);
}

static void mark_disconnected(robomesh_client_t *c) {
    pthread_mutex_lock(&c->mutex);
    c->connected = false;
    if (c->sock >= 0) {
        shutdown(c->sock, SHUT_RDWR);
        close(c->sock);
        c->sock = -1;
    }
    c->rbuf_pos = 0;
    c->rbuf_len = 0;
    pthread_mutex_unlock(&c->mutex);
}

robomesh_client_t *robomesh_client_create(const robomesh_config_t *config) {
    if (!config || !config->uuid || config->uuid[0] == '\0' ||
        !config->host || !config->keypair)
        return NULL;

    robomesh_client_t *c = calloc(1, sizeof(*c));
    if (!c) return NULL;

    strncpy(c->uuid, config->uuid, sizeof(c->uuid) - 1);
    strncpy(c->host, config->host, sizeof(c->host) - 1);
    c->tcp_port = config->tcp_port > 0 ? config->tcp_port : 5000;
    if (config->device_type)
        strncpy(c->device_type, config->device_type, sizeof(c->device_type) - 1);
    memcpy(&c->keypair, config->keypair, sizeof(robomesh_keypair_t));
    c->sock = -1;
    c->connected = false;
    c->heartbeat_seq = (int64_t)time(NULL);
    pthread_mutex_init(&c->mutex, NULL);
    return c;
}

void robomesh_client_destroy(robomesh_client_t *client) {
    if (!client) return;
    robomesh_disconnect(client);
    pthread_mutex_destroy(&client->mutex);
    /* Zero out key material */
    memset(&client->keypair, 0, sizeof(robomesh_keypair_t));
    free(client);
}

robomesh_err_t robomesh_connect(robomesh_client_t *client) {
    if (!client) return ROBOMESH_ERR_INVALID_ARG;
    if (client->connected) return ROBOMESH_OK;

    client->sock = tcp_connect(client->host, client->tcp_port);
    if (client->sock < 0) {
        set_error(client, "Failed to connect to %s:%d", client->host, client->tcp_port);
        return ROBOMESH_ERR_CONNECT;
    }
    client->connected = true;
    client->rbuf_pos = 0;
    client->rbuf_len = 0;
    return ROBOMESH_OK;
}

void robomesh_disconnect(robomesh_client_t *client) {
    if (!client) return;

    /* Signal threads to stop */
    client->hb_running = false;
    client->recv_running = false;

    /* Close socket (unblocks any blocking recv in threads) */
    mark_disconnected(client);

    /* Join threads after socket is closed */
    if (client->hb_started) {
        pthread_join(client->hb_thread, NULL);
        client->hb_started = false;
    }
    if (client->recv_started) {
        pthread_join(client->recv_thread_handle, NULL);
        client->recv_started = false;
    }
}

robomesh_err_t robomesh_reconnect(robomesh_client_t *client) {
    if (!client) return ROBOMESH_ERR_INVALID_ARG;
    robomesh_disconnect(client);
    return robomesh_connect(client);
}

bool robomesh_is_connected(const robomesh_client_t *client) {
    return client && client->connected;
}

const char *robomesh_get_jwt(const robomesh_client_t *client) {
    if (!client || client->jwt[0] == '\0') return NULL;
    return client->jwt;
}

const char *robomesh_last_error(const robomesh_client_t *client) {
    if (!client) return "NULL client";
    return client->last_error;
}

/* ── AUTH flow ────────────────────────────────────────────── */

robomesh_err_t robomesh_authenticate(robomesh_client_t *client) {
    if (!client) return ROBOMESH_ERR_INVALID_ARG;

    robomesh_err_t err;
    if (!client->connected) {
        err = robomesh_connect(client);
        if (err != ROBOMESH_OK) return err;
    }

    char buf[MAX_LINE];

    /* Step 1: Send AUTH */
    if (send_line(client->sock, "AUTH") < 0) {
        set_error(client, "Failed to send AUTH");
        mark_disconnected(client);
        return ROBOMESH_ERR_SEND;
    }

    /* Step 2: Receive AUTH_CHALLENGE */
    if (recv_line_buffered(client, buf, sizeof(buf)) < 0) {
        set_error(client, "Failed to receive AUTH_CHALLENGE");
        mark_disconnected(client);
        return ROBOMESH_ERR_RECV;
    }
    if (strcmp(buf, "AUTH_CHALLENGE") != 0) {
        set_error(client, "Expected AUTH_CHALLENGE, got: %s", buf);
        return ROBOMESH_ERR_AUTH;
    }

    /* Step 3: Send UUID */
    if (send_line(client->sock, client->uuid) < 0) {
        set_error(client, "Failed to send UUID");
        mark_disconnected(client);
        return ROBOMESH_ERR_SEND;
    }

    /* Step 4: Receive NONCE */
    if (recv_line_buffered(client, buf, sizeof(buf)) < 0) {
        set_error(client, "Failed to receive NONCE");
        mark_disconnected(client);
        return ROBOMESH_ERR_RECV;
    }
    if (strncmp(buf, "NONCE ", 6) != 0) {
        set_error(client, "Expected NONCE, got: %s", buf);
        return ROBOMESH_ERR_AUTH;
    }

    /* Step 5: Sign the nonce */
    const char *nonce_hex = buf + 6;
    uint8_t nonce_bytes[256];
    int nonce_len = hex_to_bytes(nonce_hex, nonce_bytes, sizeof(nonce_bytes));
    if (nonce_len < 0) {
        set_error(client, "Invalid nonce hex");
        return ROBOMESH_ERR_AUTH;
    }

    uint8_t sig[64];
    size_t sig_len = sizeof(sig);
    err = ed25519_sign(&client->keypair, nonce_bytes, nonce_len, sig, &sig_len);
    if (err != ROBOMESH_OK) {
        set_error(client, "Failed to sign nonce");
        return err;
    }

    char sig_hex[129];
    bytes_to_hex(sig, sig_len, sig_hex);

    /* Step 6: Send signature */
    if (send_line(client->sock, sig_hex) < 0) {
        set_error(client, "Failed to send signature");
        mark_disconnected(client);
        return ROBOMESH_ERR_SEND;
    }

    /* Step 7: Receive AUTH_OK <JWT> */
    if (recv_line_buffered(client, buf, sizeof(buf)) < 0) {
        set_error(client, "Failed to receive AUTH_OK");
        mark_disconnected(client);
        return ROBOMESH_ERR_RECV;
    }
    if (strncmp(buf, "AUTH_OK ", 8) != 0) {
        set_error(client, "Auth failed: %s", buf);
        return ROBOMESH_ERR_AUTH;
    }

    strncpy(client->jwt, buf + 8, JWT_MAX - 1);
    return ROBOMESH_OK;
}

/* ── REGISTER flow ────────────────────────────────────────── */

robomesh_err_t robomesh_register(robomesh_client_t *client, int timeout_secs) {
    if (!client) return ROBOMESH_ERR_INVALID_ARG;
    if (client->device_type[0] == '\0') {
        set_error(client, "device_type required for registration");
        return ROBOMESH_ERR_INVALID_ARG;
    }
    if (!is_valid_device_type(client->device_type)) {
        set_error(client, "Invalid device_type: must match [a-zA-Z0-9_-]{1,64}");
        return ROBOMESH_ERR_INVALID_ARG;
    }

    if (timeout_secs <= 0) timeout_secs = MAX_REGISTER_TIMEOUT;
    if (timeout_secs > MAX_REGISTER_TIMEOUT) timeout_secs = MAX_REGISTER_TIMEOUT;

    robomesh_err_t err;
    if (!client->connected) {
        err = robomesh_connect(client);
        if (err != ROBOMESH_OK) return err;
    }

    char buf[MAX_LINE];

    if (send_line(client->sock, "REGISTER") < 0) {
        set_error(client, "Failed to send REGISTER");
        mark_disconnected(client);
        return ROBOMESH_ERR_SEND;
    }

    if (recv_line_buffered(client, buf, sizeof(buf)) < 0 ||
        strcmp(buf, "REGISTER_CHALLENGE") != 0) {
        set_error(client, "Expected REGISTER_CHALLENGE, got: %s", buf);
        return ROBOMESH_ERR_AUTH;
    }

    if (send_line(client->sock, client->uuid) < 0) {
        set_error(client, "Failed to send UUID");
        mark_disconnected(client);
        return ROBOMESH_ERR_SEND;
    }

    if (recv_line_buffered(client, buf, sizeof(buf)) < 0 ||
        strcmp(buf, "SEND_DEVICE_TYPE") != 0) {
        set_error(client, "Expected SEND_DEVICE_TYPE, got: %s", buf);
        return ROBOMESH_ERR_AUTH;
    }

    if (send_line(client->sock, client->device_type) < 0) {
        set_error(client, "Failed to send device_type");
        mark_disconnected(client);
        return ROBOMESH_ERR_SEND;
    }

    if (recv_line_buffered(client, buf, sizeof(buf)) < 0 ||
        strcmp(buf, "SEND_PUBLIC_KEY") != 0) {
        set_error(client, "Expected SEND_PUBLIC_KEY, got: %s", buf);
        return ROBOMESH_ERR_AUTH;
    }

    char pub_hex[65];
    robomesh_public_key_hex(&client->keypair, pub_hex);
    if (send_line(client->sock, pub_hex) < 0) {
        set_error(client, "Failed to send public key");
        mark_disconnected(client);
        return ROBOMESH_ERR_SEND;
    }

    if (recv_line_buffered(client, buf, sizeof(buf)) < 0 ||
        strcmp(buf, "REGISTER_PENDING") != 0) {
        set_error(client, "Expected REGISTER_PENDING, got: %s", buf);
        return ROBOMESH_ERR_AUTH;
    }

    /* Wait for approval with timeout */
    struct timeval tv = { .tv_sec = timeout_secs, .tv_usec = 0 };
    setsockopt(client->sock, SOL_SOCKET, SO_RCVTIMEO, &tv, sizeof(tv));

    if (recv_line_buffered(client, buf, sizeof(buf)) < 0) {
        set_error(client, "Timed out waiting for registration response");
        return ROBOMESH_ERR_TIMEOUT;
    }

    /* Restore default timeout */
    tv.tv_sec = 30;
    setsockopt(client->sock, SOL_SOCKET, SO_RCVTIMEO, &tv, sizeof(tv));

    if (strncmp(buf, "REGISTER_OK ", 12) == 0) {
        strncpy(client->jwt, buf + 12, JWT_MAX - 1);
        return ROBOMESH_OK;
    }
    if (strcmp(buf, "REGISTER_REJECTED") == 0) {
        set_error(client, "Registration rejected");
        return ROBOMESH_ERR_AUTH;
    }

    set_error(client, "Unexpected response: %s", buf);
    return ROBOMESH_ERR_AUTH;
}

/* ── PERSIST ──────────────────────────────────────────────── */

robomesh_err_t robomesh_persist(robomesh_client_t *client) {
    if (!client || !client->connected) return ROBOMESH_ERR_INVALID_ARG;

    char buf[MAX_LINE];
    if (send_line(client->sock, "PERSIST") < 0) {
        set_error(client, "Failed to send PERSIST");
        mark_disconnected(client);
        return ROBOMESH_ERR_SEND;
    }
    if (recv_line_buffered(client, buf, sizeof(buf)) < 0) {
        set_error(client, "Failed to receive PERSIST response");
        mark_disconnected(client);
        return ROBOMESH_ERR_RECV;
    }
    if (strncmp(buf, "PERSIST_OK", 10) != 0) {
        set_error(client, "Persist failed: %s", buf);
        return ROBOMESH_ERR_AUTH;
    }
    return ROBOMESH_OK;
}

/* ── Heartbeat ────────────────────────────────────────────── */

robomesh_err_t robomesh_send_heartbeat(robomesh_client_t *client,
                                         const char *extra_data_json,
                                         int ttl) {
    if (!client) return ROBOMESH_ERR_INVALID_ARG;

    /* Atomically increment sequence number */
    pthread_mutex_lock(&client->mutex);
    client->heartbeat_seq++;
    int64_t seq = client->heartbeat_seq;
    pthread_mutex_unlock(&client->mutex);

    /* Build payload JSON */
    char payload[4096];
    int off = snprintf(payload, sizeof(payload), "{\"seq\":%lld", (long long)seq);
    if (ttl > 0)
        off += snprintf(payload + off, sizeof(payload) - off, ",\"ttl\":%d", ttl);
    if (extra_data_json)
        off += snprintf(payload + off, sizeof(payload) - off, ",\"extra_data\":%s", extra_data_json);
    int tail = snprintf(payload + off, sizeof(payload) - off, "}");
    if (off + tail >= (int)sizeof(payload)) {
        set_error(client, "Heartbeat payload too large (truncated at %zu bytes)", sizeof(payload));
        return ROBOMESH_ERR_INVALID_ARG;
    }

    /* Sign the raw JSON bytes */
    uint8_t sig[64];
    size_t sig_len = sizeof(sig);
    robomesh_err_t err = ed25519_sign(&client->keypair,
                                        (const uint8_t *)payload, strlen(payload),
                                        sig, &sig_len);
    if (err != ROBOMESH_OK) {
        set_error(client, "Failed to sign heartbeat");
        return err;
    }

    char sig_hex[129];
    bytes_to_hex(sig, sig_len, sig_hex);

    /* Build the full HEARTBEAT line */
    char line[MAX_LINE];
    snprintf(line, sizeof(line), "HEARTBEAT %s %s %s", client->uuid, payload, sig_hex);

    /* Send on a fresh TCP connection with short timeout */
    int sock = tcp_connect_with_timeout(client->host, client->tcp_port,
                                         HEARTBEAT_CONNECT_TIMEOUT);
    if (sock < 0) {
        set_error(client, "Failed to connect for heartbeat");
        return ROBOMESH_ERR_CONNECT;
    }

    int result = ROBOMESH_OK;
    if (send_line(sock, line) < 0) {
        set_error(client, "Failed to send heartbeat");
        result = ROBOMESH_ERR_SEND;
    } else {
        char buf[256];
        if (recv_line_raw(sock, buf, sizeof(buf)) < 0) {
            set_error(client, "Failed to receive heartbeat response");
            result = ROBOMESH_ERR_RECV;
        } else if (strcmp(buf, "HEARTBEAT_OK") != 0) {
            set_error(client, "Heartbeat failed: %s", buf);
            result = ROBOMESH_ERR_HEARTBEAT;
        }
    }

    close(sock);
    return result;
}

/* Background heartbeat thread */
static void *heartbeat_thread_func(void *arg) {
    robomesh_client_t *c = (robomesh_client_t *)arg;
    while (c->hb_running) {
        robomesh_send_heartbeat(c, NULL, c->hb_ttl);
        /* Sleep in 100ms increments so we can check the stop flag */
        for (int i = 0; i < c->hb_interval * 10 && c->hb_running; i++)
            usleep(100000);
    }
    return NULL;
}

robomesh_err_t robomesh_start_heartbeat(robomesh_client_t *client,
                                         int interval_secs, int ttl) {
    if (!client || interval_secs <= 0) return ROBOMESH_ERR_INVALID_ARG;
    if (client->hb_started) return ROBOMESH_OK;

    client->hb_interval = interval_secs;
    client->hb_ttl = ttl;
    client->hb_running = true;

    if (pthread_create(&client->hb_thread, NULL, heartbeat_thread_func, client) != 0) {
        client->hb_running = false;
        set_error(client, "Failed to create heartbeat thread");
        return ROBOMESH_ERR_ALLOC;
    }
    client->hb_started = true;
    return ROBOMESH_OK;
}

void robomesh_stop_heartbeat(robomesh_client_t *client) {
    if (!client || !client->hb_started) return;
    client->hb_running = false;
    pthread_join(client->hb_thread, NULL);
    client->hb_started = false;
}

/* ── Messaging ────────────────────────────────────────────── */

robomesh_err_t robomesh_send(robomesh_client_t *client, const char *message) {
    if (!client || !client->connected || !message) return ROBOMESH_ERR_INVALID_ARG;
    if (send_line(client->sock, message) < 0) {
        set_error(client, "Failed to send message");
        mark_disconnected(client);
        return ROBOMESH_ERR_SEND;
    }
    return ROBOMESH_OK;
}

robomesh_err_t robomesh_recv(robomesh_client_t *client, char *buf, size_t buf_size,
                               int timeout_ms) {
    if (!client || !client->connected || !buf) return ROBOMESH_ERR_INVALID_ARG;

    /* Only poll if no data is already buffered */
    if (timeout_ms > 0 && client->rbuf_pos >= client->rbuf_len) {
        struct pollfd pfd = { .fd = client->sock, .events = POLLIN };
        int ret = poll(&pfd, 1, timeout_ms);
        if (ret == 0) return ROBOMESH_ERR_TIMEOUT;
        if (ret < 0) {
            set_error(client, "poll error: %s", strerror(errno));
            return ROBOMESH_ERR_RECV;
        }
    }

    if (recv_line_buffered(client, buf, buf_size) < 0) {
        set_error(client, "Failed to receive");
        mark_disconnected(client);
        return ROBOMESH_ERR_RECV;
    }
    return ROBOMESH_OK;
}

/* Background receive thread */
static void *recv_thread_func(void *arg) {
    robomesh_client_t *c = (robomesh_client_t *)arg;
    char buf[MAX_LINE];
    while (c->recv_running && c->connected) {
        /* Only poll if buffer is empty */
        if (c->rbuf_pos >= c->rbuf_len) {
            struct pollfd pfd = { .fd = c->sock, .events = POLLIN };
            int ret = poll(&pfd, 1, 100); /* 100ms to check stop flag */
            if (ret == 0) continue;
            if (ret < 0 || !c->recv_running) break;
        }
        if (recv_line_buffered(c, buf, sizeof(buf)) < 0) {
            mark_disconnected(c);
            break;
        }
        if (c->recv_cb)
            c->recv_cb(buf, c->recv_cb_data);
    }
    c->recv_running = false;
    return NULL;
}

robomesh_err_t robomesh_start_recv(robomesh_client_t *client,
                                    robomesh_message_cb callback,
                                    void *user_data) {
    if (!client || !callback || !client->connected) return ROBOMESH_ERR_INVALID_ARG;
    if (client->recv_started) return ROBOMESH_OK;

    client->recv_cb = callback;
    client->recv_cb_data = user_data;
    client->recv_running = true;

    if (pthread_create(&client->recv_thread_handle, NULL, recv_thread_func, client) != 0) {
        client->recv_running = false;
        set_error(client, "Failed to create receive thread");
        return ROBOMESH_ERR_ALLOC;
    }
    client->recv_started = true;
    return ROBOMESH_OK;
}

void robomesh_stop_recv(robomesh_client_t *client) {
    if (!client || !client->recv_started) return;
    client->recv_running = false;
    /* Shutdown read side to unblock any blocking recv() */
    pthread_mutex_lock(&client->mutex);
    if (client->sock >= 0)
        shutdown(client->sock, SHUT_RD);
    pthread_mutex_unlock(&client->mutex);
    pthread_join(client->recv_thread_handle, NULL);
    client->recv_started = false;
}
