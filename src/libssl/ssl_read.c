#include <linux/types.h>

#include "bpf/bpf_helpers.h"
#include "bpf/bpf_tracing.h"

SEC("uprobe/SSL_read")
int probe_ssl_read(struct pt_regs *ctx) {
  bpf_printk("SSL_read() called\n");
  return 0;
}

SEC("uprobe/SSL_write")
int probe_ssl_write(struct pt_regs *ctx) {
  bpf_printk("SSL_write() called\n");
  return 0;
}

char _license[4] SEC("license") = "GPL";
