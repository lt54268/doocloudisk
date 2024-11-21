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
    1: i8 Ret;
    2: string Msg;
    3: list<common.File> Data;
}

struct OfficeUploadReq {
    1: string Id (api.query="id"); 
    2: string Status (api.query="status");
    3: string Key (api.query="key");
    4: string Url (api.query="url");
}

struct OfficeUploadResp {
    1: string error;
}

struct SaveReq {
    1: string Id (api.query="id"); 
    2: string Content (api.query="content");
}

struct SaveResp {
    1: i8 Ret;
    2: string Msg;
    3: common.FileContent Data;
}

struct DownloadReq {
    1: i32 FileId (api.query="fileId");
}

struct DownloadResp {
    1: i8 Ret;
    2: string Msg;
    3: string FileName;
    4: i32 FileSize;
    5: string FileContentType;
    6: string FileContent;
}

struct RemoveReq {
    1: i32 FileId (api.query="fileId");
}

struct RemoveResp {
    1: i8 Ret;
    2: string Msg;
}

service AliyunService {
    UploadResp upload(1: UploadReq request) (api.post="/api/file/content/upload");
    OfficeUploadResp office_upload(1: OfficeUploadReq request) (api.post="/api/file/content/office");
    SaveResp save(1: SaveReq request) (api.post="/api/file/content/save");
    DownloadResp download(1: DownloadReq request) (api.get="/api/file/content/download");
    RemoveResp remove(1: RemoveReq request) (api.delete="/api/file/content/remove");
}
