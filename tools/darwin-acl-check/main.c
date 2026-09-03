/*
 * main.c -- descriptor-bound macOS extended-ACL guard shared by zeroned and
 * frontier-intake.
 *
 * The object to inspect is inherited as standard input.  No pathname is
 * accepted, so a rename or symlink swap cannot redirect the authorization
 * decision.  Keep this protocol deliberately tiny: callers must match both
 * the exit status and the exact stdout line.
 */

#include <sys/acl.h>
#include <sys/stat.h>

#include <errno.h>
#include <stddef.h>
#include <string.h>
#include <unistd.h>

#define EXIT_ACL_PRESENT 10
#define EXIT_INSPECTION_ERROR 70

static const char CLEAR_RESULT[] = "zerone-darwin-acl-v1 clear\n";
static const char PRESENT_RESULT[] = "zerone-darwin-acl-v1 present\n";

static int write_all(int fd, const char *value, size_t length) {
  size_t offset = 0;

  while (offset < length) {
    ssize_t written = write(fd, value + offset, length - offset);
    if (written < 0) {
      if (errno == EINTR) {
        continue;
      }
      return -1;
    }
    if (written == 0) {
      return -1;
    }
    offset += (size_t)written;
  }
  return 0;
}

static void write_inspection_error(const char *operation, int error_number) {
  static const char prefix[] = "zerone-darwin-acl-v1 error: ";
  static const char separator[] = ": ";
  static const char newline[] = "\n";
  const char *description = strerror(error_number);

  (void)write_all(STDERR_FILENO, prefix, sizeof(prefix) - 1U);
  (void)write_all(STDERR_FILENO, operation, strlen(operation));
  (void)write_all(STDERR_FILENO, separator, sizeof(separator) - 1U);
  (void)write_all(STDERR_FILENO, description, strlen(description));
  (void)write_all(STDERR_FILENO, newline, sizeof(newline) - 1U);
}

int main(int argc, char **argv) {
  struct stat status;
  acl_t acl;
  int saved_errno;

  (void)argv;
  if (argc != 1) {
    write_inspection_error("unexpected arguments", EINVAL);
    return EXIT_INSPECTION_ERROR;
  }

  if (fstat(STDIN_FILENO, &status) != 0) {
    saved_errno = errno;
    write_inspection_error("fstat", saved_errno);
    return EXIT_INSPECTION_ERROR;
  }
  if (!S_ISREG(status.st_mode) && !S_ISDIR(status.st_mode)) {
    write_inspection_error("stdin is not a regular file or directory", EINVAL);
    return EXIT_INSPECTION_ERROR;
  }

  errno = 0;
  acl = acl_get_fd_np(STDIN_FILENO, ACL_TYPE_EXTENDED);
  if (acl != NULL) {
    if (acl_free(acl) != 0) {
      saved_errno = errno;
      write_inspection_error("acl_free", saved_errno);
      return EXIT_INSPECTION_ERROR;
    }
    if (write_all(STDOUT_FILENO, PRESENT_RESULT,
                  sizeof(PRESENT_RESULT) - 1U) != 0) {
      return EXIT_INSPECTION_ERROR;
    }
    return EXIT_ACL_PRESENT;
  }

  saved_errno = errno;
  if (saved_errno != ENOENT) {
    write_inspection_error("acl_get_fd_np", saved_errno);
    return EXIT_INSPECTION_ERROR;
  }
  if (write_all(STDOUT_FILENO, CLEAR_RESULT, sizeof(CLEAR_RESULT) - 1U) != 0) {
    return EXIT_INSPECTION_ERROR;
  }
  return 0;
}
