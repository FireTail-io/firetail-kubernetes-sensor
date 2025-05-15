#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>

#define TLS_MASK 0x100000000ULL

struct {
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 1 << 24);
} events SEC(".maps");

struct ssl_event_t {
    __u64 pid_tgid;
    __u64 ssl_ptr;
    __u64 buffer;
    int num;
};

SEC("uprobe/SSL_read")
int ssl_read_enter_v3(void *ssl, void *buffer, int num) {
    struct ssl_event_t *event;
    event = bpf_ringbuf_reserve(&events, sizeof(*event), 0);
    if (!event) return 0;

    event->pid_tgid = bpf_get_current_pid_tgid();
    event->ssl_ptr = (unsigned long)ssl;
    event->buffer = (unsigned long)buffer;
    event->num = num;

    bpf_ringbuf_submit(event, 0);
    return 0;
}

char LICENSE[] SEC("license") = "Dual BSD/GPL";