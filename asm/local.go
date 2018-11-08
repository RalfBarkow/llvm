// Problems to solve.
//
// phi instructions can reference local variables defined in basic blocks not
// yet visited when translating basic blocks in linear order.
//
// Terminator instructions can reference basic blocks not yet visited when
// translating basic blocks in linear order.
//
// The function parameters, basic blocks and local variables (produced by the
// result of instructions) of a function may be unnamed. They are assigned the
// first unused local ID (e.g. %42) when traversing the body of the function in
// linear order; where function parameters are assigned first, then for each
// basic block, assign an ID to the basic block and then to the result of its
// instructions. Note, instructions that produce void results are ignored.
// Non-value instructions (e.g. store) are always ignored. Notably, the call
// instruction may be ignored if the callee has a void return.

// TODO: make concurrent :)

package asm

import (
	"fmt"
	"strconv"

	"github.com/llir/ll/ast"
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
	"github.com/pkg/errors"
)

// funcGen is a generator for a given IR function.
type funcGen struct {
	// Module generator.
	gen *generator
	// LLVM IR function being generated.
	f *ir.Function
	// ls maps from local identifier (without '%' prefix) to corresponding IR
	// value.
	ls map[string]value.Value
}

// newFuncGen returns a new generator for the given IR function.
func newFuncGen(gen *generator, f *ir.Function) *funcGen {
	return &funcGen{
		gen: gen,
		f:   f,
		ls:  make(map[string]value.Value),
	}
}

// resolveLocals resolves the local va1riables, basic blocks and function
// parameters of the given function body. The returned value maps from local
// identifier (without '%' prefix) to the corresponding IR value.
func (fgen *funcGen) resolveLocals(body ast.FuncBody) error {
	// Create instructions (without bodies), in preparation for index.
	oldBlocks := body.Blocks()
	if err := fgen.indexLocals(oldBlocks); err != nil {
		return errors.WithStack(err)
	}
	// Translate instructions.
	f := fgen.f
	for i, block := range f.Blocks {
		insts := oldBlocks[i].Insts()
		for j, inst := range block.Insts {
			old := insts[j]
			if _, err := fgen.astToIRInst(inst, old); err != nil {
				return errors.WithStack(err)
			}
		}
	}
	// Translate terminators.
	for i, block := range f.Blocks {
		old := oldBlocks[i].Term()
		if err := fgen.astToIRTerm(block.Term, old); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

// newIRInst returns a new IR instruction (without body but with type) based on
// the given AST instruction.
func (fgen *funcGen) newIRInst(old ast.Instruction) (ir.Instruction, error) {
	switch old := old.(type) {
	// Value instructions.
	case *ast.LocalDefInst:
		ident := localIdent(old.Name())
		return fgen.newIRValueInst(ident, old.Inst())
	case ast.ValueInstruction:
		return fgen.newIRValueInst("", old)
	// Non-value instructions.
	case *ast.StoreInst:
		return &ir.InstStore{}, nil
	case *ast.FenceInst:
		return &ir.InstFence{}, nil
	default:
		panic(fmt.Errorf("support for AST instruction type %T not yet implemented", old))
	}
}

// newIRValueInst returns a new IR value instruction (without body but with
// type) based on the given AST value instruction.
func (fgen *funcGen) newIRValueInst(name string, old ast.ValueInstruction) (ir.Instruction, error) {
	switch old := old.(type) {
	// Binary instructions
	case *ast.AddInst:
		typ, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstAdd{LocalName: name, Typ: typ}, nil
	case *ast.FAddInst:
		typ, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstFAdd{LocalName: name, Typ: typ}, nil
	case *ast.SubInst:
		typ, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstSub{LocalName: name, Typ: typ}, nil
	case *ast.FSubInst:
		typ, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstFSub{LocalName: name, Typ: typ}, nil
	case *ast.MulInst:
		typ, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstMul{LocalName: name, Typ: typ}, nil
	case *ast.FMulInst:
		typ, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstFMul{LocalName: name, Typ: typ}, nil
	case *ast.UDivInst:
		typ, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstUDiv{LocalName: name, Typ: typ}, nil
	case *ast.SDivInst:
		typ, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstSDiv{LocalName: name, Typ: typ}, nil
	case *ast.FDivInst:
		typ, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstFDiv{LocalName: name, Typ: typ}, nil
	case *ast.URemInst:
		typ, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstURem{LocalName: name, Typ: typ}, nil
	case *ast.SRemInst:
		typ, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstSRem{LocalName: name, Typ: typ}, nil
	case *ast.FRemInst:
		typ, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstFRem{LocalName: name, Typ: typ}, nil
	// Bitwise instructions
	case *ast.ShlInst:
		typ, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstShl{LocalName: name, Typ: typ}, nil
	case *ast.LShrInst:
		typ, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstLShr{LocalName: name, Typ: typ}, nil
	case *ast.AShrInst:
		typ, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstAShr{LocalName: name, Typ: typ}, nil
	case *ast.AndInst:
		typ, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstAnd{LocalName: name, Typ: typ}, nil
	case *ast.OrInst:
		typ, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstOr{LocalName: name, Typ: typ}, nil
	case *ast.XorInst:
		typ, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstXor{LocalName: name, Typ: typ}, nil
	// Vector instructions
	case *ast.ExtractElementInst:
		xType, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		t, ok := xType.(*types.VectorType)
		if !ok {
			panic(fmt.Errorf("invalid vector type; expected *types.VectorType, got %T", xType))
		}
		return &ir.InstExtractElement{LocalName: name, Typ: t.ElemType}, nil
	case *ast.InsertElementInst:
		xType, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		t, ok := xType.(*types.VectorType)
		if !ok {
			panic(fmt.Errorf("invalid vector type; expected *types.VectorType, got %T", xType))
		}
		return &ir.InstInsertElement{LocalName: name, Typ: t}, nil
	case *ast.ShuffleVectorInst:
		xType, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		xt, ok := xType.(*types.VectorType)
		if !ok {
			panic(fmt.Errorf("invalid vector type; expected *types.VectorType, got %T", xType))
		}
		maskType, err := fgen.gen.irType(old.Mask().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		mt, ok := maskType.(*types.VectorType)
		if !ok {
			panic(fmt.Errorf("invalid vector type; expected *types.VectorType, got %T", maskType))
		}
		typ := types.NewVector(mt.Len, xt.ElemType)
		return &ir.InstShuffleVector{LocalName: name, Typ: typ}, nil
	// Aggregate instructions
	case *ast.ExtractValueInst:
		xType, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		indices := uintSlice(old.Indices())
		typ := aggregateElemType(xType, indices)
		return &ir.InstExtractValue{LocalName: name, Typ: typ}, nil
	case *ast.InsertValueInst:
		typ, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstInsertValue{LocalName: name, Typ: typ}, nil
	// Memory instructions
	case *ast.AllocaInst:
		elemType, err := fgen.gen.irType(old.ElemType())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		i := &ir.InstAlloca{LocalName: name, ElemType: elemType}
		// Cache i.Typ.
		i.Type()
		return i, nil
	case *ast.LoadInst:
		elemType, err := fgen.gen.irType(old.ElemType())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstLoad{LocalName: name, Typ: elemType}, nil
	case *ast.CmpXchgInst:
		oldType, err := fgen.gen.irType(old.New().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		typ := types.NewStruct(oldType, types.I8)
		return &ir.InstCmpXchg{LocalName: name, Typ: typ}, nil
	case *ast.AtomicRMWInst:
		dstType, err := fgen.gen.irType(old.Dst().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		t, ok := dstType.(*types.PointerType)
		if !ok {
			panic(fmt.Errorf("invalid pointer type; expected *types.PointerType, got %T", dstType))
		}
		return &ir.InstAtomicRMW{LocalName: name, Typ: t.ElemType}, nil
	case *ast.GetElementPtrInst:
		// TODO: handle address space of Src?
		elemType, err := fgen.gen.irType(old.ElemType())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		typ := gepType(elemType, old.Indices())
		return &ir.InstGetElementPtr{LocalName: name, Typ: typ}, nil
	// Conversion instructions
	case *ast.TruncInst:
		to, err := fgen.gen.irType(old.To())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstTrunc{LocalName: name, To: to}, nil
	case *ast.ZExtInst:
		to, err := fgen.gen.irType(old.To())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstZExt{LocalName: name, To: to}, nil
	case *ast.SExtInst:
		to, err := fgen.gen.irType(old.To())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstSExt{LocalName: name, To: to}, nil
	case *ast.FPTruncInst:
		to, err := fgen.gen.irType(old.To())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstFPTrunc{LocalName: name, To: to}, nil
	case *ast.FPExtInst:
		to, err := fgen.gen.irType(old.To())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstFPExt{LocalName: name, To: to}, nil
	case *ast.FPToUIInst:
		to, err := fgen.gen.irType(old.To())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstFPToUI{LocalName: name, To: to}, nil
	case *ast.FPToSIInst:
		to, err := fgen.gen.irType(old.To())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstFPToSI{LocalName: name, To: to}, nil
	case *ast.UIToFPInst:
		to, err := fgen.gen.irType(old.To())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstUIToFP{LocalName: name, To: to}, nil
	case *ast.SIToFPInst:
		to, err := fgen.gen.irType(old.To())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstSIToFP{LocalName: name, To: to}, nil
	case *ast.PtrToIntInst:
		to, err := fgen.gen.irType(old.To())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstPtrToInt{LocalName: name, To: to}, nil
	case *ast.IntToPtrInst:
		to, err := fgen.gen.irType(old.To())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstIntToPtr{LocalName: name, To: to}, nil
	case *ast.BitCastInst:
		to, err := fgen.gen.irType(old.To())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstBitCast{LocalName: name, To: to}, nil
	case *ast.AddrSpaceCastInst:
		to, err := fgen.gen.irType(old.To())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstAddrSpaceCast{LocalName: name, To: to}, nil
	// Other instructions
	case *ast.ICmpInst:
		xType, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		var typ types.Type
		switch xType := xType.(type) {
		case *types.IntType, *types.PointerType:
			typ = types.I1
		case *types.VectorType:
			typ = types.NewVector(xType.Len, types.I1)
		default:
			panic(fmt.Errorf("invalid icmp operand type; expected *types.IntType, *types.PointerType or *types.VectorType, got %T", xType))
		}
		return &ir.InstICmp{LocalName: name, Typ: typ}, nil
	case *ast.FCmpInst:
		xType, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		var typ types.Type
		switch xType := xType.(type) {
		case *types.FloatType:
			typ = types.I1
		case *types.VectorType:
			typ = types.NewVector(xType.Len, types.I1)
		default:
			panic(fmt.Errorf("invalid fcmp operand type; expected *types.FloatType or *types.VectorType, got %T", xType))
		}
		return &ir.InstFCmp{LocalName: name, Typ: typ}, nil
	case *ast.PhiInst:
		typ, err := fgen.gen.irType(old.Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstPhi{LocalName: name, Typ: typ}, nil
	case *ast.SelectInst:
		typ, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstSelect{LocalName: name, Typ: typ}, nil
	case *ast.CallInst:
		// NOTE: We need to store the type of call instructions before invoking
		// f.AssignIDs, since call instructions may be value instructions or
		// non-value instructions based on return type.
		typ, err := fgen.gen.irType(old.Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstCall{LocalName: name, Typ: typ}, nil
	case *ast.VAArgInst:
		argType, err := fgen.gen.irType(old.ArgType())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstVAArg{LocalName: name, ArgType: argType}, nil
	case *ast.LandingPadInst:
		resultType, err := fgen.gen.irType(old.ResultType())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstLandingPad{LocalName: name, ResultType: resultType}, nil
	case *ast.CatchPadInst:
		// Result type is always token.
		return &ir.InstCatchPad{LocalName: name}, nil
	case *ast.CleanupPadInst:
		// Result type is always token.
		return &ir.InstCleanupPad{LocalName: name}, nil
	default:
		panic(fmt.Errorf("support for AST value instruction type %T not yet implemented", old))
	}
}

// === [ Instructions ] ========================================================

// astToIRInst translates the AST instruction into an equivalent IR instruction.
func (fgen *funcGen) astToIRInst(inst ir.Instruction, old ast.Instruction) (ir.Instruction, error) {
	switch old := old.(type) {
	// Value instruction.
	case *ast.LocalDefInst:
		ident := localIdent(old.Name())
		v, ok := fgen.ls[ident]
		if !ok {
			return nil, errors.Errorf("unable to locate local identifier %q", ident)
		}
		i, ok := v.(ir.Instruction)
		if !ok {
			return nil, errors.Errorf("invalid instruction type of %q; expected ir.Instruction, got %T", ident, v)
		}
		return fgen.astToIRValueInst(i, old.Inst())
	case ast.ValueInstruction:
		return fgen.astToIRValueInst(inst, old)
	// Non-value instructions.
	case *ast.StoreInst:
		return fgen.astToIRInstStore(inst, old)
	case *ast.FenceInst:
		return fgen.astToIRInstFence(inst, old)
	default:
		panic(fmt.Errorf("support for instruction type %T not yet implemented", old))
	}
}

// astToIRValueInst translates the AST value instruction into an equivalent IR
// value instruction.
func (fgen *funcGen) astToIRValueInst(inst ir.Instruction, old ast.ValueInstruction) (ir.Instruction, error) {
	switch old := old.(type) {
	// Binary instructions
	case *ast.AddInst:
		return fgen.astToIRInstAdd(inst, old)
	case *ast.FAddInst:
		return fgen.astToIRInstFAdd(inst, old)
	case *ast.SubInst:
		return fgen.astToIRInstSub(inst, old)
	case *ast.FSubInst:
		return fgen.astToIRInstFSub(inst, old)
	case *ast.MulInst:
		return fgen.astToIRInstMul(inst, old)
	case *ast.FMulInst:
		return fgen.astToIRInstFMul(inst, old)
	case *ast.UDivInst:
		return fgen.astToIRInstUDiv(inst, old)
	case *ast.SDivInst:
		return fgen.astToIRInstSDiv(inst, old)
	case *ast.FDivInst:
		return fgen.astToIRInstFDiv(inst, old)
	case *ast.URemInst:
		return fgen.astToIRInstURem(inst, old)
	case *ast.SRemInst:
		return fgen.astToIRInstSRem(inst, old)
	case *ast.FRemInst:
		return fgen.astToIRInstFRem(inst, old)
	// Bitwise instructions
	case *ast.ShlInst:
		return fgen.astToIRInstShl(inst, old)
	case *ast.LShrInst:
		return fgen.astToIRInstLShr(inst, old)
	case *ast.AShrInst:
		return fgen.astToIRInstAShr(inst, old)
	case *ast.AndInst:
		return fgen.astToIRInstAnd(inst, old)
	case *ast.OrInst:
		return fgen.astToIRInstOr(inst, old)
	case *ast.XorInst:
		return fgen.astToIRInstXor(inst, old)
	// Vector instructions
	case *ast.ExtractElementInst:
		return fgen.astToIRInstExtractElement(inst, old)
	case *ast.InsertElementInst:
		return fgen.astToIRInstInsertElement(inst, old)
	case *ast.ShuffleVectorInst:
		return fgen.astToIRInstShuffleVector(inst, old)
	// Aggregate instructions
	case *ast.ExtractValueInst:
		return fgen.astToIRInstExtractValue(inst, old)
	case *ast.InsertValueInst:
		return fgen.astToIRInstInsertValue(inst, old)
	// Memory instructions
	case *ast.AllocaInst:
		return fgen.astToIRInstAlloca(inst, old)
	case *ast.LoadInst:
		return fgen.astToIRInstLoad(inst, old)
	case *ast.CmpXchgInst:
		return fgen.astToIRInstCmpXchg(inst, old)
	case *ast.AtomicRMWInst:
		return fgen.astToIRInstAtomicRMW(inst, old)
	case *ast.GetElementPtrInst:
		return fgen.astToIRInstGetElementPtr(inst, old)
	// Conversion instructions
	case *ast.TruncInst:
		return fgen.astToIRInstTrunc(inst, old)
	case *ast.ZExtInst:
		return fgen.astToIRInstZExt(inst, old)
	case *ast.SExtInst:
		return fgen.astToIRInstSExt(inst, old)
	case *ast.FPTruncInst:
		return fgen.astToIRInstFPTrunc(inst, old)
	case *ast.FPExtInst:
		return fgen.astToIRInstFPExt(inst, old)
	case *ast.FPToUIInst:
		return fgen.astToIRInstFPToUI(inst, old)
	case *ast.FPToSIInst:
		return fgen.astToIRInstFPToSI(inst, old)
	case *ast.UIToFPInst:
		return fgen.astToIRInstUIToFP(inst, old)
	case *ast.SIToFPInst:
		return fgen.astToIRInstSIToFP(inst, old)
	case *ast.PtrToIntInst:
		return fgen.astToIRInstPtrToInt(inst, old)
	case *ast.IntToPtrInst:
		return fgen.astToIRInstIntToPtr(inst, old)
	case *ast.BitCastInst:
		return fgen.astToIRInstBitCast(inst, old)
	case *ast.AddrSpaceCastInst:
		return fgen.astToIRInstAddrSpaceCast(inst, old)
	// Other instructions
	case *ast.ICmpInst:
		return fgen.astToIRInstICmp(inst, old)
	case *ast.FCmpInst:
		return fgen.astToIRInstFCmp(inst, old)
	case *ast.PhiInst:
		return fgen.astToIRInstPhi(inst, old)
	case *ast.SelectInst:
		return fgen.astToIRInstSelect(inst, old)
	case *ast.CallInst:
		return fgen.astToIRInstCall(inst, old)
	case *ast.VAArgInst:
		return fgen.astToIRInstVAArg(inst, old)
	case *ast.LandingPadInst:
		return fgen.astToIRInstLandingPad(inst, old)
	case *ast.CatchPadInst:
		return fgen.astToIRInstCatchPad(inst, old)
	case *ast.CleanupPadInst:
		return fgen.astToIRInstCleanupPad(inst, old)
	default:
		panic(fmt.Errorf("support for value instruction type %T not yet implemented", old))
	}
}

// ### [ Helper functions ] ####################################################

// NOTE: aggregateElemType is copied from llir/llvm/ir/inst_aggregate.go and the
// type of indicies is updated from []int64 to []uint64

// aggregateElemType returns the element type at the position in the aggregate
// type specified by the given indices.
func aggregateElemType(t types.Type, indices []uint64) types.Type {
	// Base case.
	if len(indices) == 0 {
		return t
	}
	switch t := t.(type) {
	case *types.ArrayType:
		return aggregateElemType(t.ElemType, indices[1:])
	case *types.StructType:
		return aggregateElemType(t.Fields[indices[0]], indices[1:])
	default:
		panic(fmt.Errorf("support for aggregate type %T not yet implemented", t))
	}
}

// gepType returns the pointer type to the element at the position in the type
// specified by the given indices, as calculated by the getelementptr
// instruction.
func gepType(elemType types.Type, indices []ast.TypeValue) *types.PointerType {
	e := elemType
	for i, index := range indices {
		if i == 0 {
			// Ignore checking the 0th index as it simply follows the pointer of
			// src.
			//
			// ref: http://llvm.org/docs/GetElementPtr.html#why-is-the-extra-0-index-required
			continue
		}
		switch t := e.(type) {
		case *types.PointerType:
			// ref: http://llvm.org/docs/GetElementPtr.html#what-is-dereferenced-by-gep
			panic(fmt.Errorf("unable to index into element of pointer type `%v`; for more information, see http://llvm.org/docs/GetElementPtr.html#what-is-dereferenced-by-gep", elemType))
		case *types.VectorType:
			e = t.ElemType
		case *types.ArrayType:
			e = t.ElemType
		case *types.StructType:
			idx, ok := index.Val().(*ast.IntConst)
			if !ok {
				panic(fmt.Errorf("invalid index type for structure element; expected *ast.IntConst, got %T", index))
			}
			i, err := strconv.ParseInt(idx.Text(), 10, 64)
			if err != nil {
				panic(fmt.Errorf("unable to parse integer %q; %v", idx.Text(), err))
			}
			e = t.Fields[i]
		default:
			panic(fmt.Errorf("support for indexing element type %T not yet implemented", e))
		}
	}
	return types.NewPointer(e)
}
