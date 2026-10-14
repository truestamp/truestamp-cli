// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0
//
// Records every non-loopback connect(), connectx(), sendto() and sendmsg()
// destination, every loopback one to a port below the ephemeral range,
// every one to the system DNS resolver socket and every getaddrinfo() of a
// name other than localhost, with the process path and argv, to
// $NETREC_LOG. The sandbox still refuses the call; this only says who
// tried, so a test that swallows its error still shows up. It is loaded
// through DYLD_INSERT_LIBRARIES, which macOS strips from SIP-protected
// binaries such as /bin/sh and from everything they start: such a process
// is refused but not recorded. `task test-nonet` builds it with
// `clang -dynamiclib -O2 -o netrec.dylib netrec.c`.
#include <arpa/inet.h>
#include <fcntl.h>
#include <libproc.h>
#include <netinet/in.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/socket.h>
#include <sys/un.h>
#include <unistd.h>
#include <netdb.h>
#include <crt_externs.h>

#define DYLD_INTERPOSE(_r, _o) \
  __attribute__((used)) static struct { const void *r; const void *o; } \
  _interpose_##_o __attribute__((section("__DATA,__interpose"))) = {(const void *)(unsigned long)&_r, (const void *)(unsigned long)&_o};

static void rec(const char *what) {
  const char *p = getenv("NETREC_LOG");
  if (!p) return;
  char path[PROC_PIDPATHINFO_MAXSIZE] = "?";
  proc_pidpath(getpid(), path, sizeof path);
  char args[800] = ""; int ac = *_NSGetArgc(); char **av = *_NSGetArgv();
  for (int i = 1; i < ac && strlen(args) < 700; i++) { strncat(args, " ", 2); strncat(args, av[i], 700 - strlen(args)); }
  char line[1600];
  int n = snprintf(line, sizeof line, "%d %s %s |%s\n", getpid(), path, what, args);
  int fd = open(p, O_WRONLY | O_APPEND | O_CREAT, 0644);
  if (fd >= 0) { write(fd, line, n); close(fd); }
}

// check records a destination a, reached through verb, when it is one
// the header names. A null a (a send on a connected socket) is not
// recorded here: its connect() was.
static void check(const char *verb, const struct sockaddr *a) {
  char buf[INET6_ADDRSTRLEN + 16];
  if (a && a->sa_family == AF_INET) {
    const struct sockaddr_in *i = (const void *)a;
    if ((ntohl(i->sin_addr.s_addr) >> 24) == 127 && ntohs(i->sin_port) < 49152) {
      char w[80]; snprintf(w, sizeof w, "loopback-fixed-port 127.x:%d %s", ntohs(i->sin_port), verb); rec(w);
    } else if ((ntohl(i->sin_addr.s_addr) >> 24) != 127) {
      inet_ntop(AF_INET, &i->sin_addr, buf, sizeof buf);
      char w[128]; snprintf(w, sizeof w, "%s %s:%d", verb, buf, ntohs(i->sin_port)); rec(w);
    }
  } else if (a && a->sa_family == AF_INET6) {
    const struct sockaddr_in6 *i = (const void *)a;
    if (IN6_IS_ADDR_LOOPBACK(&i->sin6_addr) && ntohs(i->sin6_port) < 49152) {
      char w[80]; snprintf(w, sizeof w, "loopback-fixed-port [::1]:%d %s", ntohs(i->sin6_port), verb); rec(w);
    } else if (!IN6_IS_ADDR_LOOPBACK(&i->sin6_addr) && !(IN6_IS_ADDR_V4MAPPED(&i->sin6_addr) && i->sin6_addr.s6_addr[12] == 127)) {
      inet_ntop(AF_INET6, &i->sin6_addr, buf, sizeof buf);
      char w[128]; snprintf(w, sizeof w, "%s [%s]:%d", verb, buf, ntohs(i->sin6_port)); rec(w);
    }
  } else if (a && a->sa_family == AF_UNIX) {
    const struct sockaddr_un *u = (const void *)a;
    if (strstr(u->sun_path, "mDNSResponder")) { char w[64]; snprintf(w, sizeof w, "%s unix mDNSResponder", verb); rec(w); }
  }
}

static int my_connect(int s, const struct sockaddr *a, socklen_t l) {
  check("connect", a);
  return connect(s, a, l);
}
DYLD_INTERPOSE(my_connect, connect)

static int my_connectx(int s, const sa_endpoints_t *e, sae_associd_t id, unsigned int f,
                       const struct iovec *iov, unsigned int n, size_t *len, sae_connid_t *cid) {
  if (e) check("connectx", e->sae_dstaddr);
  return connectx(s, e, id, f, iov, n, len, cid);
}
DYLD_INTERPOSE(my_connectx, connectx)

static ssize_t my_sendto(int s, const void *b, size_t n, int f, const struct sockaddr *a, socklen_t l) {
  check("sendto", a);
  return sendto(s, b, n, f, a, l);
}
DYLD_INTERPOSE(my_sendto, sendto)

static ssize_t my_sendmsg(int s, const struct msghdr *m, int f) {
  if (m) check("sendmsg", (const struct sockaddr *)m->msg_name);
  return sendmsg(s, m, f);
}
DYLD_INTERPOSE(my_sendmsg, sendmsg)

static int my_getaddrinfo(const char *h, const char *sv, const struct addrinfo *hi, struct addrinfo **r) {
  if (h && strcmp(h, "localhost") && strncmp(h, "127.", 4) && strcmp(h, "::1")) {
    char w[600]; snprintf(w, sizeof w, "getaddrinfo %s", h); rec(w);
  }
  return getaddrinfo(h, sv, hi, r);
}
DYLD_INTERPOSE(my_getaddrinfo, getaddrinfo)
