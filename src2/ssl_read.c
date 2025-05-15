#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>
#include <bpf/bpf_core_read.h>

#define MAX_BUF_SIZE 4096
#define TLS_MASK 0x4000000000000000ULL

char LICENSE[] SEC("license") = "Dual MIT/GPL";

struct event {
    __u32 pid;
    __u64 tid;
    int len;
    char buf[MAX_BUF_SIZE];
};

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 1024);
    __type(key, __u64);  // pid_tgid | TLS_MASK
    __type(value, void *);
} ssl_read_args SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_PERF_EVENT_ARRAY);
} events SEC(".maps");

// --- Helper to store buffer pointer at function entry ---
static __always_inline void ssl_uprobe_read_enter_v3(struct pt_regs *ctx, __u64 id, __u32 pid, void *ssl, void *buffer, int num, int dummy) {
    if (buffer == NULL)
        return;

    bpf_map_update_elem(&ssl_read_args, &id, &buffer, BPF_ANY);
}

// --- Helper to process the return from SSL_read ---
static __always_inline void process_exit_of_syscalls_read_recvfrom(struct pt_regs *ctx, __u64 id, __u64 pid, int ret, int is_tls) {
    void **bufp = bpf_map_lookup_elem(&ssl_read_args, &id);
    if (!bufp)
        return;

    void *buf = *bufp;
    bpf_map_delete_elem(&ssl_read_args, &id);

    if (ret <= 0 || ret > MAX_BUF_SIZE)
        return;

    struct event evt = {};
    evt.pid = pid;
    evt.tid = id;
    evt.len = ret;

    // Read plaintext data from buffer
    bpf_probe_read_user(&evt.buf, ret, buf);

    // Submit to userspace
    bpf_perf_event_output(ctx, &events, BPF_F_CURRENT_CPU, &evt, sizeof(evt));
}

// --- Entry probe for SSL_read ---
SEC("uprobe/SSL_read_v3")
void BPF_UPROBE(ssl_read_enter_v3, void *ssl, void *buffer, int num) {
    __u64 pid_tgid = bpf_get_current_pid_tgid();
    __u32 pid = pid_tgid >> 32;
    __u64 id = pid_tgid | TLS_MASK;

    ssl_uprobe_read_enter_v3(ctx, id, pid, ssl, buffer, num, 0);
}