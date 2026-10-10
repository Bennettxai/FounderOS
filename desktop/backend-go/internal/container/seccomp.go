package container

// SeccompProfile is the custom seccomp profile for terminal containers
// It blocks syscalls that could be used for container escape:
// - mount/pivot_root/chroot: filesystem manipulation
// - setns/unshare: namespace escape
// - ptrace: process debugging/inspection
// - kernel modules: loading malicious code
// - bpf: eBPF programs (potential escape vector)
const SeccompProfile = `{
  "defaultAction": "SCMP_ACT_ALLOW",
  "architectures": [
    "SCMP_ARCH_X86_64",
    "SCMP_ARCH_X86",
    "SCMP_ARCH_AARCH64"
  ],
  "syscalls": [
    {
      "names": ["mount", "umount", "umount2", "pivot_root", "chroot"],
      "action": "SCMP_ACT_ERRNO",
      "errnoRet": 1
    },
    {
      "names": ["setns", "unshare"],
      "action": "SCMP_ACT_ERRNO",
      "errnoRet": 1
    },
    {
      "names": ["ptrace", "process_vm_readv", "process_vm_writev"],
      "action": "SCMP_ACT_ERRNO",
      "errnoRet": 1
    },
    {
      "names": ["init_module", "finit_module", "delete_module"],
      "action": "SCMP_ACT_ERRNO",
      "errnoRet": 1
    },
    {
      "names": ["kexec_load", "kexec_file_load"],
      "action": "SCMP_ACT_ERRNO",
      "errnoRet": 1
    },
    {
      "names": ["reboot", "sethostname", "setdomainname"],
      "action": "SCMP_ACT_ERRNO",
      "errnoRet": 1
    },
    {
      "names": ["acct", "swapon", "swapoff"],
      "action": "SCMP_ACT_ERRNO",
      "errnoRet": 1
    },
    {
      "names": ["keyctl", "add_key", "request_key"],
      "action": "SCMP_ACT_ERRNO",
      "errnoRet": 1
    },
    {
      "names": ["bpf", "perf_event_open"],
      "action": "SCMP_ACT_ERRNO",
      "errnoRet": 1
    },
    {
      "names": ["userfaultfd"],
      "action": "SCMP_ACT_ERRNO",
      "errnoRet": 1
    }
  ]
}`
