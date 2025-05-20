#include <linux/bpf.h>
#include <linux/ptrace.h>
#include <linux/types.h>

#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>

struct event {
  __u64 pid;
  __u64 arg1;
  __u64 arg2;
  __u64 arg3;
};

struct {
  __uint(type, BPF_MAP_TYPE_RINGBUF);
  __uint(max_entries, 1 << 24);
} events SEC(".maps");

SEC("uprobe/SSL_read")
int probe_ssl_read(struct pt_regs *ctx) {
  bpf_printk("SSL_read() called\n");

  struct event *e;
  e = bpf_ringbuf_reserve(&events, sizeof(*e), 0);
  if (!e) return 0;

  e->pid = bpf_get_current_pid_tgid() >> 32;
  e->arg1 = PT_REGS_PARM1(ctx);
  e->arg2 = PT_REGS_PARM2(ctx);
  e->arg3 = PT_REGS_PARM3(ctx);

  bpf_ringbuf_submit(e, 0);

  return 0;
}

SEC("uprobe/SSL_write")
int probe_ssl_write(struct pt_regs *ctx) {
  bpf_printk("SSL_write() called\n");
  return 0;
}

char _license[4] SEC("license") = "GPL";
