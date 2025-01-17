// Copyright 2022 Matrix Origin
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testtxnengine

import (
	"context"
	"errors"

	"github.com/matrixorigin/matrixone/pkg/sql/parsers/tree"
	"github.com/matrixorigin/matrixone/pkg/sql/plan"
	"github.com/matrixorigin/matrixone/pkg/vm/engine"
	txnengine "github.com/matrixorigin/matrixone/pkg/vm/engine/txn"
)

type Execution struct {
	ctx  context.Context
	tx   *Tx
	stmt tree.Statement
}

var _ plan.CompilerContext = new(Execution)

func (e *Execution) Cost(obj *plan.ObjectRef, expr *plan.Expr) *plan.Cost {
	return &plan.Cost{}
}

func (e *Execution) DatabaseExists(name string) bool {
	_, err := e.tx.session.env.engine.Database(
		e.ctx,
		name,
		e.tx.operator,
	)
	return err == nil
}

func (e *Execution) DefaultDatabase() string {
	return e.tx.session.currentDB
}

func (e *Execution) GetRootSql() string {
	return ""
}

func (e *Execution) GetHideKeyDef(dbName string, tableName string) *plan.ColDef {
	attrs, err := e.getTableAttrs(dbName, tableName)
	if err != nil {
		panic(err)
	}
	for i, attr := range attrs {
		if attr.IsHidden {
			return engineAttrToPlanColDef(i, attr)
		}
	}
	return nil
}

func (e *Execution) GetPrimaryKeyDef(dbName string, tableName string) (defs []*plan.ColDef) {
	attrs, err := e.getTableAttrs(dbName, tableName)
	if err != nil {
		panic(err)
	}
	for i, attr := range attrs {
		if !attr.Primary {
			continue
		}
		defs = append(defs, engineAttrToPlanColDef(i, attr))
	}
	return
}

func (e *Execution) Resolve(schemaName string, tableName string) (objRef *plan.ObjectRef, tableDef *plan.TableDef) {
	if schemaName == "" {
		schemaName = e.tx.session.currentDB
	}

	objRef = &plan.ObjectRef{
		SchemaName: schemaName,
		ObjName:    tableName,
	}

	tableDef = &plan.TableDef{
		Name: tableName,
	}

	attrs, err := e.getTableAttrs(schemaName, tableName)
	var errDBNotFound txnengine.ErrDatabaseNotFound
	if errors.As(err, &errDBNotFound) {
		return nil, nil
	}
	var errRelNotFound txnengine.ErrRelationNotFound
	if errors.As(err, &errRelNotFound) {
		return nil, nil
	}
	if err != nil {
		panic(err)
	}

	for i, attr := range attrs {

		// return hidden columns for update or detete statement
		if attr.IsHidden {
			switch e.stmt.(type) {
			case *tree.Update, *tree.Delete:
			default:
				continue
			}
		}

		tableDef.Cols = append(tableDef.Cols, engineAttrToPlanColDef(i, attr))
	}

	//TODO properties
	//TODO view

	return
}

func (e *Execution) ResolveVariable(varName string, isSystemVar bool, isGlobalVar bool) (interface{}, error) {
	panic("unimplemented")
}

func (e *Execution) getTableAttrs(dbName string, tableName string) (attrs []*engine.Attribute, err error) {
	db, err := e.tx.session.env.engine.Database(
		e.ctx,
		dbName,
		e.tx.operator,
	)
	if err != nil {
		return nil, err
	}
	table, err := db.Relation(
		e.ctx,
		tableName,
	)
	if err != nil {
		return nil, err
	}
	defs, err := table.TableDefs(e.ctx)
	if err != nil {
		return nil, err
	}
	for _, def := range defs {
		attr, ok := def.(*engine.AttributeDef)
		if !ok {
			continue
		}
		attrs = append(attrs, &attr.Attr)
	}
	return
}

func engineAttrToPlanColDef(idx int, attr *engine.Attribute) *plan.ColDef {
	return &plan.ColDef{
		Name: attr.Name,
		Typ: &plan.Type{
			Id:        int32(attr.Type.Oid),
			Nullable:  attr.Default.NullAbility,
			Width:     attr.Type.Width,
			Precision: attr.Type.Precision,
			Size:      attr.Type.Size,
			Scale:     attr.Type.Scale,
		},
		Default: attr.Default,
		Primary: attr.Primary,
		Pkidx:   int32(idx),
		Comment: attr.Comment,
	}
}
