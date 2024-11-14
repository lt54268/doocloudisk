// idl/common.thrift
namespace go common

struct File {
  1: i32 id;
  2: i32 pid;
  3: string pids;
  4: i32 cid;
  5: string name;
  6: string type;
  7: string ext;
  8: i32 size;
  9: i32 userid;
  10: i32 share;
  11: i32 pshare;
  12: i32 created_id;
  13: string created_at;
  14: string updated_at;
  15: i32 deleted_at;
  16: string full_name;
  17: i32 overwrite;
}

struct FileContent {
  1: i32 fid;
  2: string content;
  3: string text;
  4: i32 size;
  5: i32 userid;
  6: string updated_at;
  7: string created_at;
  8: i32 id;
}
