// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.

package query

import (
	"context"
	"database/sql"

	"gorm.io/gorm"

	"gorm.io/gen"

	"gorm.io/plugin/dbresolver"
)

var (
	Q           = new(Query)
	File        *file
	FileContent *fileContent
	File_User   *file_User
)

func SetDefault(db *gorm.DB, opts ...gen.DOOption) {
	*Q = *Use(db, opts...)
	File = &Q.File
	FileContent = &Q.FileContent
	File_User = &Q.File_User
}

func Use(db *gorm.DB, opts ...gen.DOOption) *Query {
	return &Query{
		db:          db,
		File:        newFile(db, opts...),
		FileContent: newFileContent(db, opts...),
		File_User:   newFile_User(db, opts...),
	}
}

type Query struct {
	db *gorm.DB

	File        file
	FileContent fileContent
	File_User   file_User
}

func (q *Query) Available() bool { return q.db != nil }

func (q *Query) clone(db *gorm.DB) *Query {
	return &Query{
		db:          db,
		File:        q.File.clone(db),
		FileContent: q.FileContent.clone(db),
		File_User:   q.File_User.clone(db),
	}
}

func (q *Query) ReadDB() *Query {
	return q.ReplaceDB(q.db.Clauses(dbresolver.Read))
}

func (q *Query) WriteDB() *Query {
	return q.ReplaceDB(q.db.Clauses(dbresolver.Write))
}

func (q *Query) ReplaceDB(db *gorm.DB) *Query {
	return &Query{
		db:          db,
		File:        q.File.replaceDB(db),
		FileContent: q.FileContent.replaceDB(db),
		File_User:   q.File_User.replaceDB(db),
	}
}

type queryCtx struct {
	File        IFileDo
	FileContent IFileContentDo
	File_User   IFile_UserDo
}

func (q *Query) WithContext(ctx context.Context) *queryCtx {
	return &queryCtx{
		File:        q.File.WithContext(ctx),
		FileContent: q.FileContent.WithContext(ctx),
		File_User:   q.File_User.WithContext(ctx),
	}
}

func (q *Query) Transaction(fc func(tx *Query) error, opts ...*sql.TxOptions) error {
	return q.db.Transaction(func(tx *gorm.DB) error { return fc(q.clone(tx)) }, opts...)
}

func (q *Query) Begin(opts ...*sql.TxOptions) *QueryTx {
	tx := q.db.Begin(opts...)
	return &QueryTx{Query: q.clone(tx), Error: tx.Error}
}

type QueryTx struct {
	*Query
	Error error
}

func (q *QueryTx) Commit() error {
	return q.db.Commit().Error
}

func (q *QueryTx) Rollback() error {
	return q.db.Rollback().Error
}

func (q *QueryTx) SavePoint(name string) error {
	return q.db.SavePoint(name).Error
}

func (q *QueryTx) RollbackTo(name string) error {
	return q.db.RollbackTo(name).Error
}
