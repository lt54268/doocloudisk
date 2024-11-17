// idl/common.thrift
namespace go common

struct File {
  1: i64 id;
  2: i64 pid;
  3: string pids;
  4: i64 cid;
  5: string name;
  6: string type;
  7: string ext;
  8: i64 size;
  9: i64 userid;
  10: i32 share;
  11: i64 pshare;
  12: i64 created_id;
  13: string created_at;
  14: string updated_at;
  15: i64 deleted_at;
  16: string full_name;
  17: i64 overwrite;
}

struct FileContent {
  1: i64 fid;
  2: string content;
  3: string text;
  4: i64 size;
  5: i64 userid;
  6: string updated_at;
  7: string created_at;
  8: i64 id;
}
