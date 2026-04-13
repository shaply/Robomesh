/**
 * Robomesh Robot SDK - MQTT client implementation.
 *
 * Topic-based protocol with challenge-response auth using libmosquitto.
 */

#include "robomesh_mqtt.h"
#include "robomesh.h"

#include <stdio.h>
#include <stdlib.h>
#include <stdarg.h>
#include <string.h>
#include <unistd.h>
#include <time.h>
#include <pthread.h>

#include <mosquitto.h>
#include <openssl/evp.h>

#define JWT_MAX 2048
#define ERR_MAX 512
#define TOPIC_MAX 256
#define PAYLOAD_MAX 8192

/* ── Internal structures ──────────────────────────────────── */

struct robomesh_mqtt_client {
    char uuid[256];
    char host[256];
    int mqtt_port;
    robomesh_keypair_t keypair;
    struct mosquitto *mosq;
    bool connected;
    char jwt[JWT_MAX];
    int64_t heartbeat_seq;
    char last_error[ERR_MAX];

    pthread_mutex_t mutex;

    /* Topic strings (pre-computed) */
    char topic_auth[TOPIC_MAX];
    char topic_auth_resp[TOPIC_MAX];
    char topic_heartbeat[TOPIC_MAX];
    char topic_heartbeat_resp[TOPIC_MAX];
    char topic_message[TOPIC_MAX];
    char topic_to_robot[TOPIC_MAX];

    /* Auth synchronization */
    char auth_response[PAYLOAD_MAX];
    pthread_mutex_t auth_mutex;
    pthread_cond_t auth_cond;
    bool auth_received;

    /* Heartbeat synchronization */
    char hb_response[PAYLOAD_MAX];
    pthread_mutex_t hb_mutex;
    pthread_cond_t hb_cond;
    bool hb_received;

    /* Message callback */
    robomesh_mqtt_message_cb msg_cb;
    void *msg_cb_data;

    /* Background heartbeat thread */
    pthread_t hb_thread;
    volatile bool hb_running;
    bool hb_started;
    int hb_interval;
    int hb_ttl;
};

/* ── Hex utilities ────────────────────────────────────────── */

static void mqtt_bytes_to_hex(const uint8_t *bytes, size_t len, char *hex) {
    for (size_t i = 0; i < len; i++)
        sprintf(hex + i * 2, "%02x", bytes[i]);
    hex[len * 2] = '\0';
}

static int mqtt_hex_to_bytes(const char *hex, uint8_t *bytes, size_t max_len) {
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

static robomesh_err_t mqtt_ed25519_sign(const robomesh_keypair_t *kp,
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

/* ── JSON helpers ────────────────────────────────────────── */

static int mqtt_json_get_string(const char *json, const char *key, char *out, size_t out_size) {
    char pattern[128];
    snprintf(pattern, sizeof(pattern), "\"%s\":\"", key);
    const char *start = strstr(json, pattern);
    if (!start) {
        snprintf(pattern, sizeof(pattern), "\"%s\": \"", key);
        start = strstr(json, pattern);
        if (!start) return -1;
    }
    start = strchr(start, ':');
    if (!start) return -1;
    start++;
    while (*start == ' ') start++;
    if (*start != '"') return -1;
    start++;

    size_t i = 0;
    while (*start && *start != '"' && i < out_size - 1) {
        if (*start == '\\' && *(start + 1))
            start++;
        out[i++] = *start++;
    }
    out[i] = '\0';
    return 0;
}

/* ── Error handling ──────────────────────────────────────── */

static void mqtt_set_error(robomesh_mqtt_client_t *c, const char *fmt, ...) {
    va_list args;
    va_start(args, fmt);
    vsnprintf(c->last_error, ERR_MAX, fmt, args);
    va_end(args);
}

/* ── Mosquitto callbacks ─────────────────────────────────── */

static void on_connect_cb(struct mosquitto *mosq, void *obj, int rc) {
    robomesh_mqtt_client_t *c = (robomesh_mqtt_client_t *)obj;
    if (rc == 0) {
        c->connected = true;
        /* Subscribe to response and incoming message topics */
        mosquitto_subscribe(mosq, NULL, c->topic_auth_resp, 0);
        mosquitto_subscribe(mosq, NULL, c->topic_heartbeat_resp, 0);
        mosquitto_subscribe(mosq, NULL, c->topic_to_robot, 0);
    }
}

static void on_disconnect_cb(struct mosquitto *mosq, void *obj, int rc) {
    robomesh_mqtt_client_t *c = (robomesh_mqtt_client_t *)obj;
    (void)mosq;
    (void)rc;
    c->connected = false;
}

static void on_message_cb(struct mosquitto *mosq, void *obj,
                           const struct mosquitto_message *msg) {
    robomesh_mqtt_client_t *c = (robomesh_mqtt_client_t *)obj;
    (void)mosq;

    if (!msg->payload || msg->payloadlen <= 0) return;

    /* Null-terminate payload */
    char payload[PAYLOAD_MAX];
    size_t len = (size_t)msg->payloadlen < sizeof(payload) - 1
                     ? (size_t)msg->payloadlen
                     : sizeof(payload) - 1;
    memcpy(payload, msg->payload, len);
    payload[len] = '\0';

    if (strcmp(msg->topic, c->topic_auth_resp) == 0) {
        pthread_mutex_lock(&c->auth_mutex);
        strncpy(c->auth_response, payload, sizeof(c->auth_response) - 1);
        c->auth_response[sizeof(c->auth_response) - 1] = '\0';
        c->auth_received = true;
        pthread_cond_signal(&c->auth_cond);
        pthread_mutex_unlock(&c->auth_mutex);
    } else if (strcmp(msg->topic, c->topic_heartbeat_resp) == 0) {
        pthread_mutex_lock(&c->hb_mutex);
        strncpy(c->hb_response, payload, sizeof(c->hb_response) - 1);
        c->hb_response[sizeof(c->hb_response) - 1] = '\0';
        c->hb_received = true;
        pthread_cond_signal(&c->hb_cond);
        pthread_mutex_unlock(&c->hb_mutex);
    } else if (strcmp(msg->topic, c->topic_to_robot) == 0) {
        if (c->msg_cb)
            c->msg_cb(payload, c->msg_cb_data);
    }
}

/* ── Client lifecycle ────────────────────────────────────── */

robomesh_mqtt_client_t *robomesh_mqtt_create(const robomesh_mqtt_config_t *config) {
    if (!config || !config->uuid || config->uuid[0] == '\0' ||
        !config->host || !config->keypair)
        return NULL;

    robomesh_mqtt_client_t *c = calloc(1, sizeof(*c));
    if (!c) return NULL;

    strncpy(c->uuid, config->uuid, sizeof(c->uuid) - 1);
    strncpy(c->host, config->host, sizeof(c->host) - 1);
    c->mqtt_port = config->mqtt_port > 0 ? config->mqtt_port : 1883;
    memcpy(&c->keypair, config->keypair, sizeof(robomesh_keypair_t));
    c->heartbeat_seq = (int64_t)time(NULL);

    /* Pre-compute topic strings */
    snprintf(c->topic_auth, TOPIC_MAX, "robomesh/auth/%s", c->uuid);
    snprintf(c->topic_auth_resp, TOPIC_MAX, "robomesh/auth/%s/response", c->uuid);
    snprintf(c->topic_heartbeat, TOPIC_MAX, "robomesh/heartbeat/%s", c->uuid);
    snprintf(c->topic_heartbeat_resp, TOPIC_MAX, "robomesh/heartbeat/%s/response", c->uuid);
    snprintf(c->topic_message, TOPIC_MAX, "robomesh/message/%s", c->uuid);
    snprintf(c->topic_to_robot, TOPIC_MAX, "robomesh/to_robot/%s", c->uuid);

    pthread_mutex_init(&c->mutex, NULL);
    pthread_mutex_init(&c->auth_mutex, NULL);
    pthread_mutex_init(&c->hb_mutex, NULL);
    pthread_cond_init(&c->auth_cond, NULL);
    pthread_cond_init(&c->hb_cond, NULL);

    /* Initialize mosquitto library (idempotent) */
    mosquitto_lib_init();

    char client_id[300];
    snprintf(client_id, sizeof(client_id), "robomesh-%s", c->uuid);
    c->mosq = mosquitto_new(client_id, true, c);
    if (!c->mosq) {
        free(c);
        return NULL;
    }

    mosquitto_connect_callback_set(c->mosq, on_connect_cb);
    mosquitto_disconnect_callback_set(c->mosq, on_disconnect_cb);
    mosquitto_message_callback_set(c->mosq, on_message_cb);

    return c;
}

void robomesh_mqtt_destroy(robomesh_mqtt_client_t *client) {
    if (!client) return;
    robomesh_mqtt_disconnect(client);
    if (client->mosq)
        mosquitto_destroy(client->mosq);
    pthread_mutex_destroy(&client->mutex);
    pthread_mutex_destroy(&client->auth_mutex);
    pthread_mutex_destroy(&client->hb_mutex);
    pthread_cond_destroy(&client->auth_cond);
    pthread_cond_destroy(&client->hb_cond);
    memset(&client->keypair, 0, sizeof(robomesh_keypair_t));
    free(client);
}

robomesh_err_t robomesh_mqtt_connect(robomesh_mqtt_client_t *client) {
    if (!client) return ROBOMESH_ERR_INVALID_ARG;

    int rc = mosquitto_connect(client->mosq, client->host, client->mqtt_port, 60);
    if (rc != MOSQ_ERR_SUCCESS) {
        mqtt_set_error(client, "Failed to connect to MQTT broker: %s",
                       mosquitto_strerror(rc));
        return ROBOMESH_ERR_CONNECT;
    }

    rc = mosquitto_loop_start(client->mosq);
    if (rc != MOSQ_ERR_SUCCESS) {
        mqtt_set_error(client, "Failed to start MQTT loop: %s",
                       mosquitto_strerror(rc));
        return ROBOMESH_ERR_CONNECT;
    }

    /* Wait for connection callback */
    for (int i = 0; i < 50 && !client->connected; i++)
        usleep(100000);

    if (!client->connected) {
        mqtt_set_error(client, "MQTT connection timed out");
        return ROBOMESH_ERR_TIMEOUT;
    }

    return ROBOMESH_OK;
}

void robomesh_mqtt_disconnect(robomesh_mqtt_client_t *client) {
    if (!client) return;

    client->hb_running = false;
    if (client->hb_started) {
        pthread_join(client->hb_thread, NULL);
        client->hb_started = false;
    }

    if (client->mosq) {
        mosquitto_loop_stop(client->mosq, false);
        mosquitto_disconnect(client->mosq);
    }
    client->connected = false;
}

bool robomesh_mqtt_is_connected(const robomesh_mqtt_client_t *client) {
    return client && client->connected;
}

const char *robomesh_mqtt_get_jwt(const robomesh_mqtt_client_t *client) {
    if (!client || client->jwt[0] == '\0') return NULL;
    return client->jwt;
}

const char *robomesh_mqtt_last_error(const robomesh_mqtt_client_t *client) {
    if (!client) return "NULL client";
    return client->last_error;
}

/* ── Timed wait helper ───────────────────────────────────── */

static int wait_for_response(pthread_mutex_t *mtx, pthread_cond_t *cond,
                              bool *flag, int timeout_ms) {
    struct timespec ts;
    clock_gettime(CLOCK_REALTIME, &ts);
    ts.tv_sec += timeout_ms / 1000;
    ts.tv_nsec += (timeout_ms % 1000) * 1000000L;
    if (ts.tv_nsec >= 1000000000L) {
        ts.tv_sec++;
        ts.tv_nsec -= 1000000000L;
    }

    pthread_mutex_lock(mtx);
    while (!*flag) {
        if (pthread_cond_timedwait(cond, mtx, &ts) != 0) {
            pthread_mutex_unlock(mtx);
            return -1; /* timeout */
        }
    }
    *flag = false;
    pthread_mutex_unlock(mtx);
    return 0;
}

/* ── AUTH flow ───────────────────────────────────────────── */

robomesh_err_t robomesh_mqtt_authenticate(robomesh_mqtt_client_t *client, int timeout_ms) {
    if (!client || !client->connected) return ROBOMESH_ERR_INVALID_ARG;
    if (timeout_ms <= 0) timeout_ms = 10000;

    char payload[PAYLOAD_MAX];

    /* Step 1: Request nonce */
    pthread_mutex_lock(&client->auth_mutex);
    client->auth_received = false;
    client->auth_response[0] = '\0';
    pthread_mutex_unlock(&client->auth_mutex);

    snprintf(payload, sizeof(payload), "{\"uuid\":\"%s\"}", client->uuid);
    mosquitto_publish(client->mosq, NULL, client->topic_auth,
                      (int)strlen(payload), payload, 0, false);

    if (wait_for_response(&client->auth_mutex, &client->auth_cond,
                          &client->auth_received, timeout_ms) < 0) {
        mqtt_set_error(client, "Auth step 1 timed out");
        return ROBOMESH_ERR_TIMEOUT;
    }

    /* Parse step 1 response */
    char status[64];
    if (mqtt_json_get_string(client->auth_response, "status", status, sizeof(status)) < 0) {
        mqtt_set_error(client, "Invalid auth response");
        return ROBOMESH_ERR_AUTH;
    }
    if (strcmp(status, "error") == 0) {
        char err_msg[256];
        mqtt_json_get_string(client->auth_response, "error", err_msg, sizeof(err_msg));
        mqtt_set_error(client, "Auth step 1: %s", err_msg);
        return ROBOMESH_ERR_AUTH;
    }
    if (strcmp(status, "nonce") != 0) {
        mqtt_set_error(client, "Expected nonce, got: %s", status);
        return ROBOMESH_ERR_AUTH;
    }

    char nonce_hex[512];
    if (mqtt_json_get_string(client->auth_response, "nonce", nonce_hex, sizeof(nonce_hex)) < 0) {
        mqtt_set_error(client, "No nonce in response");
        return ROBOMESH_ERR_AUTH;
    }

    /* Step 2: Sign nonce */
    uint8_t nonce_bytes[256];
    int nonce_len = mqtt_hex_to_bytes(nonce_hex, nonce_bytes, sizeof(nonce_bytes));
    if (nonce_len < 0) {
        mqtt_set_error(client, "Invalid nonce hex");
        return ROBOMESH_ERR_AUTH;
    }

    uint8_t sig[64];
    size_t sig_len = sizeof(sig);
    robomesh_err_t err = mqtt_ed25519_sign(&client->keypair, nonce_bytes, nonce_len,
                                            sig, &sig_len);
    if (err != ROBOMESH_OK) {
        mqtt_set_error(client, "Failed to sign nonce");
        return err;
    }

    char sig_hex[129];
    mqtt_bytes_to_hex(sig, sig_len, sig_hex);

    pthread_mutex_lock(&client->auth_mutex);
    client->auth_received = false;
    client->auth_response[0] = '\0';
    pthread_mutex_unlock(&client->auth_mutex);

    snprintf(payload, sizeof(payload),
             "{\"uuid\":\"%s\",\"signature\":\"%s\",\"nonce\":\"%s\"}",
             client->uuid, sig_hex, nonce_hex);
    mosquitto_publish(client->mosq, NULL, client->topic_auth,
                      (int)strlen(payload), payload, 0, false);

    if (wait_for_response(&client->auth_mutex, &client->auth_cond,
                          &client->auth_received, timeout_ms) < 0) {
        mqtt_set_error(client, "Auth step 2 timed out");
        return ROBOMESH_ERR_TIMEOUT;
    }

    /* Parse step 2 response */
    if (mqtt_json_get_string(client->auth_response, "status", status, sizeof(status)) < 0) {
        mqtt_set_error(client, "Invalid auth response");
        return ROBOMESH_ERR_AUTH;
    }
    if (strcmp(status, "error") == 0) {
        char err_msg[256];
        mqtt_json_get_string(client->auth_response, "error", err_msg, sizeof(err_msg));
        mqtt_set_error(client, "Auth step 2: %s", err_msg);
        return ROBOMESH_ERR_AUTH;
    }
    if (strcmp(status, "ok") != 0) {
        mqtt_set_error(client, "Expected ok, got: %s", status);
        return ROBOMESH_ERR_AUTH;
    }

    if (mqtt_json_get_string(client->auth_response, "jwt", client->jwt, JWT_MAX) < 0) {
        mqtt_set_error(client, "No JWT in auth response");
        return ROBOMESH_ERR_AUTH;
    }

    return ROBOMESH_OK;
}

/* ── Heartbeat ───────────────────────────────────────────── */

robomesh_err_t robomesh_mqtt_send_heartbeat(robomesh_mqtt_client_t *client,
                                             const char *extra_data_json,
                                             int ttl) {
    if (!client || !client->connected) return ROBOMESH_ERR_INVALID_ARG;

    pthread_mutex_lock(&client->mutex);
    client->heartbeat_seq++;
    int64_t seq = client->heartbeat_seq;
    pthread_mutex_unlock(&client->mutex);

    /* Build payload JSON string (MQTT heartbeat sends as string, not object) */
    char payload_json[4096];
    int off = snprintf(payload_json, sizeof(payload_json), "{\"seq\":%lld", (long long)seq);
    if (ttl > 0)
        off += snprintf(payload_json + off, sizeof(payload_json) - off, ",\"ttl\":%d", ttl);
    if (extra_data_json)
        off += snprintf(payload_json + off, sizeof(payload_json) - off,
                        ",\"extra_data\":%s", extra_data_json);
    int tail = snprintf(payload_json + off, sizeof(payload_json) - off, "}");
    if (off + tail >= (int)sizeof(payload_json)) {
        mqtt_set_error(client, "Heartbeat payload too large");
        return ROBOMESH_ERR_INVALID_ARG;
    }

    /* Sign the raw JSON bytes */
    uint8_t sig[64];
    size_t sig_len = sizeof(sig);
    robomesh_err_t err = mqtt_ed25519_sign(&client->keypair,
                                            (const uint8_t *)payload_json,
                                            strlen(payload_json),
                                            sig, &sig_len);
    if (err != ROBOMESH_OK) {
        mqtt_set_error(client, "Failed to sign heartbeat");
        return err;
    }

    char sig_hex[129];
    mqtt_bytes_to_hex(sig, sig_len, sig_hex);

    /* Build MQTT heartbeat message: payload is a JSON string (escaped) */
    char msg[PAYLOAD_MAX];
    /* We need to JSON-escape the payload_json string for embedding */
    char escaped_payload[8192];
    size_t ei = 0;
    for (size_t i = 0; payload_json[i] && ei < sizeof(escaped_payload) - 2; i++) {
        char ch = payload_json[i];
        if (ch == '"' || ch == '\\') {
            escaped_payload[ei++] = '\\';
        }
        escaped_payload[ei++] = ch;
    }
    escaped_payload[ei] = '\0';

    snprintf(msg, sizeof(msg),
             "{\"payload\":\"%s\",\"signature\":\"%s\"}",
             escaped_payload, sig_hex);

    pthread_mutex_lock(&client->hb_mutex);
    client->hb_received = false;
    client->hb_response[0] = '\0';
    pthread_mutex_unlock(&client->hb_mutex);

    mosquitto_publish(client->mosq, NULL, client->topic_heartbeat,
                      (int)strlen(msg), msg, 0, false);

    if (wait_for_response(&client->hb_mutex, &client->hb_cond,
                          &client->hb_received, 5000) < 0) {
        mqtt_set_error(client, "Heartbeat timed out");
        return ROBOMESH_ERR_TIMEOUT;
    }

    char status[64];
    if (mqtt_json_get_string(client->hb_response, "status", status, sizeof(status)) == 0 &&
        strcmp(status, "error") == 0) {
        char err_msg[256];
        mqtt_json_get_string(client->hb_response, "error", err_msg, sizeof(err_msg));
        mqtt_set_error(client, "Heartbeat: %s", err_msg);
        return ROBOMESH_ERR_HEARTBEAT;
    }

    return ROBOMESH_OK;
}

static void *mqtt_heartbeat_thread_func(void *arg) {
    robomesh_mqtt_client_t *c = (robomesh_mqtt_client_t *)arg;
    while (c->hb_running) {
        robomesh_mqtt_send_heartbeat(c, NULL, c->hb_ttl);
        for (int i = 0; i < c->hb_interval * 10 && c->hb_running; i++)
            usleep(100000);
    }
    return NULL;
}

robomesh_err_t robomesh_mqtt_start_heartbeat(robomesh_mqtt_client_t *client,
                                              int interval_secs, int ttl) {
    if (!client || interval_secs <= 0) return ROBOMESH_ERR_INVALID_ARG;
    if (client->hb_started) return ROBOMESH_OK;

    client->hb_interval = interval_secs;
    client->hb_ttl = ttl;
    client->hb_running = true;

    if (pthread_create(&client->hb_thread, NULL, mqtt_heartbeat_thread_func, client) != 0) {
        client->hb_running = false;
        mqtt_set_error(client, "Failed to create heartbeat thread");
        return ROBOMESH_ERR_ALLOC;
    }
    client->hb_started = true;
    return ROBOMESH_OK;
}

void robomesh_mqtt_stop_heartbeat(robomesh_mqtt_client_t *client) {
    if (!client || !client->hb_started) return;
    client->hb_running = false;
    pthread_join(client->hb_thread, NULL);
    client->hb_started = false;
}

/* ── Messaging ───────────────────────────────────────────── */

robomesh_err_t robomesh_mqtt_send(robomesh_mqtt_client_t *client, const char *message) {
    if (!client || !message || !client->connected) return ROBOMESH_ERR_INVALID_ARG;

    int rc = mosquitto_publish(client->mosq, NULL, client->topic_message,
                               (int)strlen(message), message, 0, false);
    if (rc != MOSQ_ERR_SUCCESS) {
        mqtt_set_error(client, "Failed to publish message: %s", mosquitto_strerror(rc));
        return ROBOMESH_ERR_SEND;
    }
    return ROBOMESH_OK;
}

void robomesh_mqtt_on_message(robomesh_mqtt_client_t *client,
                               robomesh_mqtt_message_cb callback,
                               void *user_data) {
    if (!client) return;
    client->msg_cb = callback;
    client->msg_cb_data = user_data;
}
