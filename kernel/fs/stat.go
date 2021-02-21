//
// fs.go
//
// Copyright (c) 2021 Markku Rossi
//
// All rights reserved.
//

package fs

const (
	S_IFMT   int = 0170000 /* type of file */
	S_IFIFO  int = 0010000 /* named pipe (fifo) */
	S_IFCHR  int = 0020000 /* character special */
	S_IFDIR  int = 0040000 /* directory */
	S_IFBLK  int = 0060000 /* block special */
	S_IFREG  int = 0100000 /* regular */
	S_IFLNK  int = 0120000 /* symbolic link */
	S_IFSOCK int = 0140000 /* socket */
	S_IFWHT  int = 0160000 /* whiteout */
)
