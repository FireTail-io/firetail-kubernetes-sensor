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

SEC("uprobe/SSL_read_v3")
void BPF_UPROBE(ssl_read_enter_v3, void* ssl, void* buffer, int num) {     
    __u64 pid_tgid = bpf_get_current_pid_tgid();
    __u32 pid = pid_tgid >> 32;
    __u64 id = pid_tgid | TLS_MASK;
    ssl_uprobe_read_enter_v3(ctx, id, pid, ssl, buffer, num, 0);
}

SEC("uretprobe/SSL_read")
void BPF_URETPROBE(ssl_ret_read_v3) {
 __u64 pid_tgid = bpf_get_current_pid_tgid();
    __u64 pid = pid_tgid >> 32;
    __u64 id = pid_tgid | TLS_MASK;

    int returnValue = PT_REGS_RC(ctx);

    process_exit_of_syscalls_read_recvfrom(ctx, id, pid, returnValue, 1);
}

char LICENSE[] SEC("license") = "Dual BSD/GPL";