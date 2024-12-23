// idl/aliyun.thrift
namespace go aliyun
include "common.thrift"
// struct File {
//   1: i32 id;
//   2: i32 pid;
//   3: string pids;
//   4: i32 cid;
//   5: string name;
//   6: string type;
//   7: string ext;
//   8: i32 size;
//   9: i32 userid;
//   10: i32 share;
//   11: i32 pshare;
//   12: i32 created_id;
//   13: string created_at;
//   14: string updated_at;
//   15: i32 deleted_at;
//   16: string full_name;
//   17: i32 overwrite;
// }

// struct FileContent {
//   1: i32 fid;
//   2: string content;
//   3: string text;
//   4: i32 size;
//   5: i32 userid;
//   6: string updated_at;
//   7: string created_at;
//   8: i32 id;
// }
struct UploadReq {
    1: string Pid (api.query="pid"); 
    2: string Cover (api.query="cover");
    3: string WebkitRelativePath (api.query="webkitRelativePath");
}

struct UploadResp {
    1: i8 ret;
    2: string msg;
    3: list<common.File> data;
}

struct IoUploadReq {
    1: string Pid (api.query="pid"); 
    2: string Cover (api.query="cover");
    3: string WebkitRelativePath (api.query="webkitRelativePath");
    4: i32 FileId (api.query="id");
}

struct IoUploadResp {
    1: i8 ret;
    2: string msg;
    3: list<common.File> data;
}

struct OfficeUploadReq {
    1: i32 Id (api.query="id", api.json="id"); 
    2: i32 Status (api.query="status", api.json="status");
    3: string Key (api.query="key", api.json="key");
    4: string Url (api.query="url", api.json="url");
}

struct OfficeUploadResp {
    1: i32 error;
}

struct SaveReq {
    1: i32 Id (api.json="id"); 
    2: string Content (api.json="content");
}

struct SaveResp {
    1: i8 ret;
    2: string msg;
    3: list<common.FileContent> data;
}

struct DownloadReq {
    1: i32 FileId (api.query="id");
}

struct DownloadResp {
    1: i8 ret;
    2: string msg;
    3: list<common.File> data;
}

struct DownloadOfficeReq {
    1: string Key (api.query="key");
}

struct DownloadOfficeResp {
    1: i8 ret;
    2: string msg;
    3: list<common.File> data;
}

struct RemoveReq {
    1: i32 FileId (api.query="id");
}

struct RemoveResp {
    1: i8 ret;
    2: string msg;
    3: list<common.File> data;
}

struct FileStatus {
    1: i32 id;
    2: string status;
}

struct StatusReq {
    1: list<i32> FileIds (api.query="ids");
}

struct StatusResp {
    1: i8 ret;
    2: string msg;
    3: list<FileStatus> data;
}

service AliyunService {
    UploadResp upload(1: UploadReq request) (api.post="/api/file/content/upload");
    IoUploadResp io_upload(1: IoUploadReq request) (api.post="/api/file/content/io_upload");
    OfficeUploadResp office_upload(1: OfficeUploadReq request) (api.get="/api/file/content/office", api.post="/api/file/content/office");
    SaveResp save(1: SaveReq request) (api.post="/api/file/content/save");
    DownloadResp download(1: DownloadReq request) (api.get="/api/file/content/download");
    DownloadResp downloading(1: DownloadReq request) (api.get="/api/file/content/downloading");
    DownloadResp downloading_office(1: DownloadOfficeReq request) (api.get="/api/file/content/downloading_office");
    RemoveResp remove(1: RemoveReq request) (api.delete="/api/file/content/remove");
    StatusResp status(1: StatusReq request) (api.get="/api/file/content/status");
}
