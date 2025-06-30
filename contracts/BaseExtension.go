// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package contracts

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// Claim is an auto generated low-level Go binding around an user-defined struct.
type Claim struct {
	Id           *big.Int
	ParentId     *big.Int
	Recipient    common.Address
	Value        *big.Int
	StateId      [32]byte
	StateData    []byte
	StrategyId   [32]byte
	StrategyData []byte
}

// WithdrawParams is an auto generated low-level Go binding around an user-defined struct.
type WithdrawParams struct {
	Amount       *big.Int
	Token        common.Address
	StrategyData []byte
	StateData    []byte
}

// BaseExtensionMetaData contains all meta data concerning the BaseExtension contract.
var BaseExtensionMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"BaseExtension_AlreadyInitialized\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"BaseExtension_InvalidFactory\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"BaseExtension_UnknownPool\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"BaseExtension_UnsupportedHook\",\"type\":\"error\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"parentId\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"stateId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"stateData\",\"type\":\"bytes\"},{\"internalType\":\"bytes32\",\"name\":\"strategyId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"strategyData\",\"type\":\"bytes\"}],\"internalType\":\"structClaim[]\",\"name\":\"claims\",\"type\":\"tuple[]\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"Token\",\"name\":\"token\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"strategyData\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"stateData\",\"type\":\"bytes\"}],\"internalType\":\"structWithdrawParams[]\",\"name\":\"params\",\"type\":\"tuple[]\"}],\"name\":\"afterBatchWithdraw\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"afterInitialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"afterInitializePool\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"parentId\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"stateId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"stateData\",\"type\":\"bytes\"},{\"internalType\":\"bytes32\",\"name\":\"strategyId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"strategyData\",\"type\":\"bytes\"}],\"internalType\":\"structClaim\",\"name\":\"claim\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"Token\",\"name\":\"token\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"strategyData\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"stateData\",\"type\":\"bytes\"}],\"internalType\":\"structWithdrawParams\",\"name\":\"params\",\"type\":\"tuple\"}],\"name\":\"afterWithdraw\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"parentId\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"stateId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"stateData\",\"type\":\"bytes\"},{\"internalType\":\"bytes32\",\"name\":\"strategyId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"strategyData\",\"type\":\"bytes\"}],\"internalType\":\"structClaim[]\",\"name\":\"claims\",\"type\":\"tuple[]\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"Token\",\"name\":\"token\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"strategyData\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"stateData\",\"type\":\"bytes\"}],\"internalType\":\"structWithdrawParams[]\",\"name\":\"params\",\"type\":\"tuple[]\"}],\"name\":\"beforeBatchWithdraw\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"beforeInitialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"beforeInitializePool\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"parentId\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"stateId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"stateData\",\"type\":\"bytes\"},{\"internalType\":\"bytes32\",\"name\":\"strategyId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"strategyData\",\"type\":\"bytes\"}],\"internalType\":\"structClaim\",\"name\":\"claim\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"Token\",\"name\":\"token\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"strategyData\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"stateData\",\"type\":\"bytes\"}],\"internalType\":\"structWithdrawParams\",\"name\":\"params\",\"type\":\"tuple\"}],\"name\":\"beforeWithdraw\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"parentId\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"stateId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"stateData\",\"type\":\"bytes\"},{\"internalType\":\"bytes32\",\"name\":\"strategyId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"strategyData\",\"type\":\"bytes\"}],\"internalType\":\"structClaim[]\",\"name\":\"claims\",\"type\":\"tuple[]\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"Token\",\"name\":\"token\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"strategyData\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"stateData\",\"type\":\"bytes\"}],\"internalType\":\"structWithdrawParams[]\",\"name\":\"params\",\"type\":\"tuple[]\"}],\"name\":\"checkBatchState\",\"outputs\":[],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"parentId\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"stateId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"stateData\",\"type\":\"bytes\"},{\"internalType\":\"bytes32\",\"name\":\"strategyId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"strategyData\",\"type\":\"bytes\"}],\"internalType\":\"structClaim\",\"name\":\"claim\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"Token\",\"name\":\"token\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"strategyData\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"stateData\",\"type\":\"bytes\"}],\"internalType\":\"structWithdrawParams\",\"name\":\"params\",\"type\":\"tuple\"}],\"name\":\"checkState\",\"outputs\":[],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"extensionId\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"extensionName\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"__factory\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"parentId\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"stateId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"stateData\",\"type\":\"bytes\"},{\"internalType\":\"bytes32\",\"name\":\"strategyId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"strategyData\",\"type\":\"bytes\"}],\"internalType\":\"structClaim\",\"name\":\"claim\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"Token\",\"name\":\"token\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"strategyData\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"stateData\",\"type\":\"bytes\"}],\"internalType\":\"structWithdrawParams\",\"name\":\"params\",\"type\":\"tuple\"}],\"name\":\"releasable\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// BaseExtensionABI is the input ABI used to generate the binding from.
// Deprecated: Use BaseExtensionMetaData.ABI instead.
var BaseExtensionABI = BaseExtensionMetaData.ABI

// BaseExtension is an auto generated Go binding around an Ethereum contract.
type BaseExtension struct {
	BaseExtensionCaller     // Read-only binding to the contract
	BaseExtensionTransactor // Write-only binding to the contract
	BaseExtensionFilterer   // Log filterer for contract events
}

// BaseExtensionCaller is an auto generated read-only Go binding around an Ethereum contract.
type BaseExtensionCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BaseExtensionTransactor is an auto generated write-only Go binding around an Ethereum contract.
type BaseExtensionTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BaseExtensionFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type BaseExtensionFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BaseExtensionSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type BaseExtensionSession struct {
	Contract     *BaseExtension    // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// BaseExtensionCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type BaseExtensionCallerSession struct {
	Contract *BaseExtensionCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts        // Call options to use throughout this session
}

// BaseExtensionTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type BaseExtensionTransactorSession struct {
	Contract     *BaseExtensionTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// BaseExtensionRaw is an auto generated low-level Go binding around an Ethereum contract.
type BaseExtensionRaw struct {
	Contract *BaseExtension // Generic contract binding to access the raw methods on
}

// BaseExtensionCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type BaseExtensionCallerRaw struct {
	Contract *BaseExtensionCaller // Generic read-only contract binding to access the raw methods on
}

// BaseExtensionTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type BaseExtensionTransactorRaw struct {
	Contract *BaseExtensionTransactor // Generic write-only contract binding to access the raw methods on
}

// NewBaseExtension creates a new instance of BaseExtension, bound to a specific deployed contract.
func NewBaseExtension(address common.Address, backend bind.ContractBackend) (*BaseExtension, error) {
	contract, err := bindBaseExtension(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &BaseExtension{BaseExtensionCaller: BaseExtensionCaller{contract: contract}, BaseExtensionTransactor: BaseExtensionTransactor{contract: contract}, BaseExtensionFilterer: BaseExtensionFilterer{contract: contract}}, nil
}

// NewBaseExtensionCaller creates a new read-only instance of BaseExtension, bound to a specific deployed contract.
func NewBaseExtensionCaller(address common.Address, caller bind.ContractCaller) (*BaseExtensionCaller, error) {
	contract, err := bindBaseExtension(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &BaseExtensionCaller{contract: contract}, nil
}

// NewBaseExtensionTransactor creates a new write-only instance of BaseExtension, bound to a specific deployed contract.
func NewBaseExtensionTransactor(address common.Address, transactor bind.ContractTransactor) (*BaseExtensionTransactor, error) {
	contract, err := bindBaseExtension(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &BaseExtensionTransactor{contract: contract}, nil
}

// NewBaseExtensionFilterer creates a new log filterer instance of BaseExtension, bound to a specific deployed contract.
func NewBaseExtensionFilterer(address common.Address, filterer bind.ContractFilterer) (*BaseExtensionFilterer, error) {
	contract, err := bindBaseExtension(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &BaseExtensionFilterer{contract: contract}, nil
}

// bindBaseExtension binds a generic wrapper to an already deployed contract.
func bindBaseExtension(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := BaseExtensionMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BaseExtension *BaseExtensionRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BaseExtension.Contract.BaseExtensionCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BaseExtension *BaseExtensionRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BaseExtension.Contract.BaseExtensionTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BaseExtension *BaseExtensionRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BaseExtension.Contract.BaseExtensionTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BaseExtension *BaseExtensionCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BaseExtension.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BaseExtension *BaseExtensionTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BaseExtension.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BaseExtension *BaseExtensionTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BaseExtension.Contract.contract.Transact(opts, method, params...)
}

// CheckBatchState is a free data retrieval call binding the contract method 0x2ad45de1.
//
// Solidity: function checkBatchState((uint256,uint256,address,uint256,bytes32,bytes,bytes32,bytes)[] claims, (uint256,address,bytes,bytes)[] params) view returns()
func (_BaseExtension *BaseExtensionCaller) CheckBatchState(opts *bind.CallOpts, claims []Claim, params []WithdrawParams) error {
	var out []interface{}
	err := _BaseExtension.contract.Call(opts, &out, "checkBatchState", claims, params)

	if err != nil {
		return err
	}

	return err

}

// CheckBatchState is a free data retrieval call binding the contract method 0x2ad45de1.
//
// Solidity: function checkBatchState((uint256,uint256,address,uint256,bytes32,bytes,bytes32,bytes)[] claims, (uint256,address,bytes,bytes)[] params) view returns()
func (_BaseExtension *BaseExtensionSession) CheckBatchState(claims []Claim, params []WithdrawParams) error {
	return _BaseExtension.Contract.CheckBatchState(&_BaseExtension.CallOpts, claims, params)
}

// CheckBatchState is a free data retrieval call binding the contract method 0x2ad45de1.
//
// Solidity: function checkBatchState((uint256,uint256,address,uint256,bytes32,bytes,bytes32,bytes)[] claims, (uint256,address,bytes,bytes)[] params) view returns()
func (_BaseExtension *BaseExtensionCallerSession) CheckBatchState(claims []Claim, params []WithdrawParams) error {
	return _BaseExtension.Contract.CheckBatchState(&_BaseExtension.CallOpts, claims, params)
}

// CheckState is a free data retrieval call binding the contract method 0x75170a32.
//
// Solidity: function checkState((uint256,uint256,address,uint256,bytes32,bytes,bytes32,bytes) claim, (uint256,address,bytes,bytes) params) view returns()
func (_BaseExtension *BaseExtensionCaller) CheckState(opts *bind.CallOpts, claim Claim, params WithdrawParams) error {
	var out []interface{}
	err := _BaseExtension.contract.Call(opts, &out, "checkState", claim, params)

	if err != nil {
		return err
	}

	return err

}

// CheckState is a free data retrieval call binding the contract method 0x75170a32.
//
// Solidity: function checkState((uint256,uint256,address,uint256,bytes32,bytes,bytes32,bytes) claim, (uint256,address,bytes,bytes) params) view returns()
func (_BaseExtension *BaseExtensionSession) CheckState(claim Claim, params WithdrawParams) error {
	return _BaseExtension.Contract.CheckState(&_BaseExtension.CallOpts, claim, params)
}

// CheckState is a free data retrieval call binding the contract method 0x75170a32.
//
// Solidity: function checkState((uint256,uint256,address,uint256,bytes32,bytes,bytes32,bytes) claim, (uint256,address,bytes,bytes) params) view returns()
func (_BaseExtension *BaseExtensionCallerSession) CheckState(claim Claim, params WithdrawParams) error {
	return _BaseExtension.Contract.CheckState(&_BaseExtension.CallOpts, claim, params)
}

// ExtensionId is a free data retrieval call binding the contract method 0x62d7076e.
//
// Solidity: function extensionId() view returns(bytes32)
func (_BaseExtension *BaseExtensionCaller) ExtensionId(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _BaseExtension.contract.Call(opts, &out, "extensionId")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// ExtensionId is a free data retrieval call binding the contract method 0x62d7076e.
//
// Solidity: function extensionId() view returns(bytes32)
func (_BaseExtension *BaseExtensionSession) ExtensionId() ([32]byte, error) {
	return _BaseExtension.Contract.ExtensionId(&_BaseExtension.CallOpts)
}

// ExtensionId is a free data retrieval call binding the contract method 0x62d7076e.
//
// Solidity: function extensionId() view returns(bytes32)
func (_BaseExtension *BaseExtensionCallerSession) ExtensionId() ([32]byte, error) {
	return _BaseExtension.Contract.ExtensionId(&_BaseExtension.CallOpts)
}

// ExtensionName is a free data retrieval call binding the contract method 0x1c86100f.
//
// Solidity: function extensionName() view returns(string)
func (_BaseExtension *BaseExtensionCaller) ExtensionName(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _BaseExtension.contract.Call(opts, &out, "extensionName")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// ExtensionName is a free data retrieval call binding the contract method 0x1c86100f.
//
// Solidity: function extensionName() view returns(string)
func (_BaseExtension *BaseExtensionSession) ExtensionName() (string, error) {
	return _BaseExtension.Contract.ExtensionName(&_BaseExtension.CallOpts)
}

// ExtensionName is a free data retrieval call binding the contract method 0x1c86100f.
//
// Solidity: function extensionName() view returns(string)
func (_BaseExtension *BaseExtensionCallerSession) ExtensionName() (string, error) {
	return _BaseExtension.Contract.ExtensionName(&_BaseExtension.CallOpts)
}

// Releasable is a free data retrieval call binding the contract method 0xc730fe30.
//
// Solidity: function releasable((uint256,uint256,address,uint256,bytes32,bytes,bytes32,bytes) claim, (uint256,address,bytes,bytes) params) view returns(uint256)
func (_BaseExtension *BaseExtensionCaller) Releasable(opts *bind.CallOpts, claim Claim, params WithdrawParams) (*big.Int, error) {
	var out []interface{}
	err := _BaseExtension.contract.Call(opts, &out, "releasable", claim, params)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Releasable is a free data retrieval call binding the contract method 0xc730fe30.
//
// Solidity: function releasable((uint256,uint256,address,uint256,bytes32,bytes,bytes32,bytes) claim, (uint256,address,bytes,bytes) params) view returns(uint256)
func (_BaseExtension *BaseExtensionSession) Releasable(claim Claim, params WithdrawParams) (*big.Int, error) {
	return _BaseExtension.Contract.Releasable(&_BaseExtension.CallOpts, claim, params)
}

// Releasable is a free data retrieval call binding the contract method 0xc730fe30.
//
// Solidity: function releasable((uint256,uint256,address,uint256,bytes32,bytes,bytes32,bytes) claim, (uint256,address,bytes,bytes) params) view returns(uint256)
func (_BaseExtension *BaseExtensionCallerSession) Releasable(claim Claim, params WithdrawParams) (*big.Int, error) {
	return _BaseExtension.Contract.Releasable(&_BaseExtension.CallOpts, claim, params)
}

// AfterBatchWithdraw is a paid mutator transaction binding the contract method 0x77d63e17.
//
// Solidity: function afterBatchWithdraw((uint256,uint256,address,uint256,bytes32,bytes,bytes32,bytes)[] claims, (uint256,address,bytes,bytes)[] params) returns()
func (_BaseExtension *BaseExtensionTransactor) AfterBatchWithdraw(opts *bind.TransactOpts, claims []Claim, params []WithdrawParams) (*types.Transaction, error) {
	return _BaseExtension.contract.Transact(opts, "afterBatchWithdraw", claims, params)
}

// AfterBatchWithdraw is a paid mutator transaction binding the contract method 0x77d63e17.
//
// Solidity: function afterBatchWithdraw((uint256,uint256,address,uint256,bytes32,bytes,bytes32,bytes)[] claims, (uint256,address,bytes,bytes)[] params) returns()
func (_BaseExtension *BaseExtensionSession) AfterBatchWithdraw(claims []Claim, params []WithdrawParams) (*types.Transaction, error) {
	return _BaseExtension.Contract.AfterBatchWithdraw(&_BaseExtension.TransactOpts, claims, params)
}

// AfterBatchWithdraw is a paid mutator transaction binding the contract method 0x77d63e17.
//
// Solidity: function afterBatchWithdraw((uint256,uint256,address,uint256,bytes32,bytes,bytes32,bytes)[] claims, (uint256,address,bytes,bytes)[] params) returns()
func (_BaseExtension *BaseExtensionTransactorSession) AfterBatchWithdraw(claims []Claim, params []WithdrawParams) (*types.Transaction, error) {
	return _BaseExtension.Contract.AfterBatchWithdraw(&_BaseExtension.TransactOpts, claims, params)
}

// AfterInitialize is a paid mutator transaction binding the contract method 0xa58b1277.
//
// Solidity: function afterInitialize(bytes data) returns()
func (_BaseExtension *BaseExtensionTransactor) AfterInitialize(opts *bind.TransactOpts, data []byte) (*types.Transaction, error) {
	return _BaseExtension.contract.Transact(opts, "afterInitialize", data)
}

// AfterInitialize is a paid mutator transaction binding the contract method 0xa58b1277.
//
// Solidity: function afterInitialize(bytes data) returns()
func (_BaseExtension *BaseExtensionSession) AfterInitialize(data []byte) (*types.Transaction, error) {
	return _BaseExtension.Contract.AfterInitialize(&_BaseExtension.TransactOpts, data)
}

// AfterInitialize is a paid mutator transaction binding the contract method 0xa58b1277.
//
// Solidity: function afterInitialize(bytes data) returns()
func (_BaseExtension *BaseExtensionTransactorSession) AfterInitialize(data []byte) (*types.Transaction, error) {
	return _BaseExtension.Contract.AfterInitialize(&_BaseExtension.TransactOpts, data)
}

// AfterInitializePool is a paid mutator transaction binding the contract method 0xef40dc09.
//
// Solidity: function afterInitializePool(bytes data) returns()
func (_BaseExtension *BaseExtensionTransactor) AfterInitializePool(opts *bind.TransactOpts, data []byte) (*types.Transaction, error) {
	return _BaseExtension.contract.Transact(opts, "afterInitializePool", data)
}

// AfterInitializePool is a paid mutator transaction binding the contract method 0xef40dc09.
//
// Solidity: function afterInitializePool(bytes data) returns()
func (_BaseExtension *BaseExtensionSession) AfterInitializePool(data []byte) (*types.Transaction, error) {
	return _BaseExtension.Contract.AfterInitializePool(&_BaseExtension.TransactOpts, data)
}

// AfterInitializePool is a paid mutator transaction binding the contract method 0xef40dc09.
//
// Solidity: function afterInitializePool(bytes data) returns()
func (_BaseExtension *BaseExtensionTransactorSession) AfterInitializePool(data []byte) (*types.Transaction, error) {
	return _BaseExtension.Contract.AfterInitializePool(&_BaseExtension.TransactOpts, data)
}

// AfterWithdraw is a paid mutator transaction binding the contract method 0x405b7950.
//
// Solidity: function afterWithdraw((uint256,uint256,address,uint256,bytes32,bytes,bytes32,bytes) claim, (uint256,address,bytes,bytes) params) returns()
func (_BaseExtension *BaseExtensionTransactor) AfterWithdraw(opts *bind.TransactOpts, claim Claim, params WithdrawParams) (*types.Transaction, error) {
	return _BaseExtension.contract.Transact(opts, "afterWithdraw", claim, params)
}

// AfterWithdraw is a paid mutator transaction binding the contract method 0x405b7950.
//
// Solidity: function afterWithdraw((uint256,uint256,address,uint256,bytes32,bytes,bytes32,bytes) claim, (uint256,address,bytes,bytes) params) returns()
func (_BaseExtension *BaseExtensionSession) AfterWithdraw(claim Claim, params WithdrawParams) (*types.Transaction, error) {
	return _BaseExtension.Contract.AfterWithdraw(&_BaseExtension.TransactOpts, claim, params)
}

// AfterWithdraw is a paid mutator transaction binding the contract method 0x405b7950.
//
// Solidity: function afterWithdraw((uint256,uint256,address,uint256,bytes32,bytes,bytes32,bytes) claim, (uint256,address,bytes,bytes) params) returns()
func (_BaseExtension *BaseExtensionTransactorSession) AfterWithdraw(claim Claim, params WithdrawParams) (*types.Transaction, error) {
	return _BaseExtension.Contract.AfterWithdraw(&_BaseExtension.TransactOpts, claim, params)
}

// BeforeBatchWithdraw is a paid mutator transaction binding the contract method 0x31526dae.
//
// Solidity: function beforeBatchWithdraw((uint256,uint256,address,uint256,bytes32,bytes,bytes32,bytes)[] claims, (uint256,address,bytes,bytes)[] params) returns()
func (_BaseExtension *BaseExtensionTransactor) BeforeBatchWithdraw(opts *bind.TransactOpts, claims []Claim, params []WithdrawParams) (*types.Transaction, error) {
	return _BaseExtension.contract.Transact(opts, "beforeBatchWithdraw", claims, params)
}

// BeforeBatchWithdraw is a paid mutator transaction binding the contract method 0x31526dae.
//
// Solidity: function beforeBatchWithdraw((uint256,uint256,address,uint256,bytes32,bytes,bytes32,bytes)[] claims, (uint256,address,bytes,bytes)[] params) returns()
func (_BaseExtension *BaseExtensionSession) BeforeBatchWithdraw(claims []Claim, params []WithdrawParams) (*types.Transaction, error) {
	return _BaseExtension.Contract.BeforeBatchWithdraw(&_BaseExtension.TransactOpts, claims, params)
}

// BeforeBatchWithdraw is a paid mutator transaction binding the contract method 0x31526dae.
//
// Solidity: function beforeBatchWithdraw((uint256,uint256,address,uint256,bytes32,bytes,bytes32,bytes)[] claims, (uint256,address,bytes,bytes)[] params) returns()
func (_BaseExtension *BaseExtensionTransactorSession) BeforeBatchWithdraw(claims []Claim, params []WithdrawParams) (*types.Transaction, error) {
	return _BaseExtension.Contract.BeforeBatchWithdraw(&_BaseExtension.TransactOpts, claims, params)
}

// BeforeInitialize is a paid mutator transaction binding the contract method 0x971b7aa7.
//
// Solidity: function beforeInitialize(bytes data) returns()
func (_BaseExtension *BaseExtensionTransactor) BeforeInitialize(opts *bind.TransactOpts, data []byte) (*types.Transaction, error) {
	return _BaseExtension.contract.Transact(opts, "beforeInitialize", data)
}

// BeforeInitialize is a paid mutator transaction binding the contract method 0x971b7aa7.
//
// Solidity: function beforeInitialize(bytes data) returns()
func (_BaseExtension *BaseExtensionSession) BeforeInitialize(data []byte) (*types.Transaction, error) {
	return _BaseExtension.Contract.BeforeInitialize(&_BaseExtension.TransactOpts, data)
}

// BeforeInitialize is a paid mutator transaction binding the contract method 0x971b7aa7.
//
// Solidity: function beforeInitialize(bytes data) returns()
func (_BaseExtension *BaseExtensionTransactorSession) BeforeInitialize(data []byte) (*types.Transaction, error) {
	return _BaseExtension.Contract.BeforeInitialize(&_BaseExtension.TransactOpts, data)
}

// BeforeInitializePool is a paid mutator transaction binding the contract method 0x19bbe29c.
//
// Solidity: function beforeInitializePool(bytes data) returns()
func (_BaseExtension *BaseExtensionTransactor) BeforeInitializePool(opts *bind.TransactOpts, data []byte) (*types.Transaction, error) {
	return _BaseExtension.contract.Transact(opts, "beforeInitializePool", data)
}

// BeforeInitializePool is a paid mutator transaction binding the contract method 0x19bbe29c.
//
// Solidity: function beforeInitializePool(bytes data) returns()
func (_BaseExtension *BaseExtensionSession) BeforeInitializePool(data []byte) (*types.Transaction, error) {
	return _BaseExtension.Contract.BeforeInitializePool(&_BaseExtension.TransactOpts, data)
}

// BeforeInitializePool is a paid mutator transaction binding the contract method 0x19bbe29c.
//
// Solidity: function beforeInitializePool(bytes data) returns()
func (_BaseExtension *BaseExtensionTransactorSession) BeforeInitializePool(data []byte) (*types.Transaction, error) {
	return _BaseExtension.Contract.BeforeInitializePool(&_BaseExtension.TransactOpts, data)
}

// BeforeWithdraw is a paid mutator transaction binding the contract method 0x5f6df45e.
//
// Solidity: function beforeWithdraw((uint256,uint256,address,uint256,bytes32,bytes,bytes32,bytes) claim, (uint256,address,bytes,bytes) params) returns()
func (_BaseExtension *BaseExtensionTransactor) BeforeWithdraw(opts *bind.TransactOpts, claim Claim, params WithdrawParams) (*types.Transaction, error) {
	return _BaseExtension.contract.Transact(opts, "beforeWithdraw", claim, params)
}

// BeforeWithdraw is a paid mutator transaction binding the contract method 0x5f6df45e.
//
// Solidity: function beforeWithdraw((uint256,uint256,address,uint256,bytes32,bytes,bytes32,bytes) claim, (uint256,address,bytes,bytes) params) returns()
func (_BaseExtension *BaseExtensionSession) BeforeWithdraw(claim Claim, params WithdrawParams) (*types.Transaction, error) {
	return _BaseExtension.Contract.BeforeWithdraw(&_BaseExtension.TransactOpts, claim, params)
}

// BeforeWithdraw is a paid mutator transaction binding the contract method 0x5f6df45e.
//
// Solidity: function beforeWithdraw((uint256,uint256,address,uint256,bytes32,bytes,bytes32,bytes) claim, (uint256,address,bytes,bytes) params) returns()
func (_BaseExtension *BaseExtensionTransactorSession) BeforeWithdraw(claim Claim, params WithdrawParams) (*types.Transaction, error) {
	return _BaseExtension.Contract.BeforeWithdraw(&_BaseExtension.TransactOpts, claim, params)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address __factory) returns()
func (_BaseExtension *BaseExtensionTransactor) Initialize(opts *bind.TransactOpts, __factory common.Address) (*types.Transaction, error) {
	return _BaseExtension.contract.Transact(opts, "initialize", __factory)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address __factory) returns()
func (_BaseExtension *BaseExtensionSession) Initialize(__factory common.Address) (*types.Transaction, error) {
	return _BaseExtension.Contract.Initialize(&_BaseExtension.TransactOpts, __factory)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address __factory) returns()
func (_BaseExtension *BaseExtensionTransactorSession) Initialize(__factory common.Address) (*types.Transaction, error) {
	return _BaseExtension.Contract.Initialize(&_BaseExtension.TransactOpts, __factory)
}
