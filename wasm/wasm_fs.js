//
// Copyright (c) 2021 Markku Rossi
//
// All rights reserved.
//

global.fs = {
    constants: { O_WRONLY: -1, O_RDWR: -1, O_CREAT: -1, O_TRUNC: -1, O_APPEND: -1, O_EXCL: -1 }, // unused
    write(fd, buf, offset, length, position, callback) {
        if (position != null) {
	    callback(enosys());
	    return;
        }
        syscall_write(fd, buf, offset, length, callback);
        return;
    },
    chmod(path, mode, callback) { callback(enosys()); },
    chown(path, uid, gid, callback) { callback(enosys()); },
    close(fd, callback) { callback(enosys()); },
    fchmod(fd, mode, callback) { callback(enosys()); },
    fchown(fd, uid, gid, callback) { callback(enosys()); },
    fstat(fd, callback) { callback(enosys()); },
    fsync(fd, callback) { callback(null); },
    ftruncate(fd, length, callback) { callback(enosys()); },
    lchown(path, uid, gid, callback) { callback(enosys()); },
    link(path, link, callback) { callback(enosys()); },
    lstat(path, callback) { callback(enosys()); },
    mkdir(path, perm, callback) { callback(enosys()); },
    open(path, flags, mode, callback) { callback(enosys()); },
    read(fd, buffer, offset, length, position, callback) { callback(enosys()); },
    readdir(path, callback) { callback(enosys()); },
    readlink(path, callback) { callback(enosys()); },
    rename(from, to, callback) { callback(enosys()); },
    rmdir(path, callback) { callback(enosys()); },
    stat(path, callback) { callback(enosys()); },
    symlink(path, link, callback) { callback(enosys()); },
    truncate(path, length, callback) { callback(enosys()); },
    unlink(path, callback) { callback(enosys()); },
    utimes(path, atime, mtime, callback) { callback(enosys()); },
};
