package main

type nfsShareInfo struct {
	NfsServerFqdn string `json:"nfsServerFqdn"`
	NfsServerPath string `json:"nfsServerPath"`
	MountPoint    string `json:"mountPoint"`
	FreeSpace     uint64 `json:"freeSpace"`
	TotalSpace    uint64 `json:"usedSpace"`
	FolderCount   int    `json:"folderCount"`
}

type nfsShareResult struct {
	NfsPath   string `json:"nfsPath"`
	IsEnabled string `json:"isEnabled"`
	MountPath string `json:"mountPath"`
}
