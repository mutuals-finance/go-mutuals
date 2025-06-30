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

// PoolFactoryMetaData contains all meta data concerning the PoolFactory contract.
var PoolFactoryMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\"}],\"name\":\"AddressEmptyCode\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"implementation\",\"type\":\"address\"}],\"name\":\"ERC1967InvalidImplementation\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ERC1967NonPayable\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"FailedCall\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidInitialization\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotInitializing\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"OwnableInvalidOwner\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"OwnableUnauthorizedAccount\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"UUPSUnauthorizedCallContext\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"slot\",\"type\":\"bytes32\"}],\"name\":\"UUPSUnsupportedProxiableUUID\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"version\",\"type\":\"uint64\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"pool\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"registry\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bytes32[]\",\"name\":\"extensions\",\"type\":\"bytes32[]\"},{\"indexed\":false,\"internalType\":\"bytes[]\",\"name\":\"data\",\"type\":\"bytes[]\"}],\"name\":\"PoolCreated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"implementation\",\"type\":\"address\"}],\"name\":\"Upgraded\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"UPGRADE_INTERFACE_VERSION\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_owner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_beacon\",\"type\":\"address\"}],\"name\":\"__PoolFactory_init\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"beacon\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_initialOwner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_registry\",\"type\":\"address\"},{\"internalType\":\"bytes32[]\",\"name\":\"_extensions\",\"type\":\"bytes32[]\"},{\"internalType\":\"bytes[]\",\"name\":\"_data\",\"type\":\"bytes[]\"},{\"internalType\":\"uint256\",\"name\":\"_salt\",\"type\":\"uint256\"}],\"name\":\"createPool\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"pool\",\"type\":\"address\"}],\"name\":\"created\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_initialOwner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_registry\",\"type\":\"address\"},{\"internalType\":\"bytes32[]\",\"name\":\"_extensions\",\"type\":\"bytes32[]\"},{\"internalType\":\"bytes[]\",\"name\":\"_data\",\"type\":\"bytes[]\"},{\"internalType\":\"uint256\",\"name\":\"_salt\",\"type\":\"uint256\"}],\"name\":\"getAddress\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"proxiableUUID\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newImplementation\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"upgradeToAndCall\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"}]",
}

// PoolFactoryABI is the input ABI used to generate the binding from.
// Deprecated: Use PoolFactoryMetaData.ABI instead.
var PoolFactoryABI = PoolFactoryMetaData.ABI

// PoolFactory is an auto generated Go binding around an Ethereum contract.
type PoolFactory struct {
	PoolFactoryCaller     // Read-only binding to the contract
	PoolFactoryTransactor // Write-only binding to the contract
	PoolFactoryFilterer   // Log filterer for contract events
}

// PoolFactoryCaller is an auto generated read-only Go binding around an Ethereum contract.
type PoolFactoryCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PoolFactoryTransactor is an auto generated write-only Go binding around an Ethereum contract.
type PoolFactoryTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PoolFactoryFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type PoolFactoryFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PoolFactorySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type PoolFactorySession struct {
	Contract     *PoolFactory      // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// PoolFactoryCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type PoolFactoryCallerSession struct {
	Contract *PoolFactoryCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts      // Call options to use throughout this session
}

// PoolFactoryTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type PoolFactoryTransactorSession struct {
	Contract     *PoolFactoryTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts      // Transaction auth options to use throughout this session
}

// PoolFactoryRaw is an auto generated low-level Go binding around an Ethereum contract.
type PoolFactoryRaw struct {
	Contract *PoolFactory // Generic contract binding to access the raw methods on
}

// PoolFactoryCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type PoolFactoryCallerRaw struct {
	Contract *PoolFactoryCaller // Generic read-only contract binding to access the raw methods on
}

// PoolFactoryTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type PoolFactoryTransactorRaw struct {
	Contract *PoolFactoryTransactor // Generic write-only contract binding to access the raw methods on
}

// NewPoolFactory creates a new instance of PoolFactory, bound to a specific deployed contract.
func NewPoolFactory(address common.Address, backend bind.ContractBackend) (*PoolFactory, error) {
	contract, err := bindPoolFactory(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &PoolFactory{PoolFactoryCaller: PoolFactoryCaller{contract: contract}, PoolFactoryTransactor: PoolFactoryTransactor{contract: contract}, PoolFactoryFilterer: PoolFactoryFilterer{contract: contract}}, nil
}

// NewPoolFactoryCaller creates a new read-only instance of PoolFactory, bound to a specific deployed contract.
func NewPoolFactoryCaller(address common.Address, caller bind.ContractCaller) (*PoolFactoryCaller, error) {
	contract, err := bindPoolFactory(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &PoolFactoryCaller{contract: contract}, nil
}

// NewPoolFactoryTransactor creates a new write-only instance of PoolFactory, bound to a specific deployed contract.
func NewPoolFactoryTransactor(address common.Address, transactor bind.ContractTransactor) (*PoolFactoryTransactor, error) {
	contract, err := bindPoolFactory(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &PoolFactoryTransactor{contract: contract}, nil
}

// NewPoolFactoryFilterer creates a new log filterer instance of PoolFactory, bound to a specific deployed contract.
func NewPoolFactoryFilterer(address common.Address, filterer bind.ContractFilterer) (*PoolFactoryFilterer, error) {
	contract, err := bindPoolFactory(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &PoolFactoryFilterer{contract: contract}, nil
}

// bindPoolFactory binds a generic wrapper to an already deployed contract.
func bindPoolFactory(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := PoolFactoryMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_PoolFactory *PoolFactoryRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _PoolFactory.Contract.PoolFactoryCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_PoolFactory *PoolFactoryRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _PoolFactory.Contract.PoolFactoryTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_PoolFactory *PoolFactoryRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _PoolFactory.Contract.PoolFactoryTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_PoolFactory *PoolFactoryCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _PoolFactory.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_PoolFactory *PoolFactoryTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _PoolFactory.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_PoolFactory *PoolFactoryTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _PoolFactory.Contract.contract.Transact(opts, method, params...)
}

// UPGRADEINTERFACEVERSION is a free data retrieval call binding the contract method 0xad3cb1cc.
//
// Solidity: function UPGRADE_INTERFACE_VERSION() view returns(string)
func (_PoolFactory *PoolFactoryCaller) UPGRADEINTERFACEVERSION(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _PoolFactory.contract.Call(opts, &out, "UPGRADE_INTERFACE_VERSION")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// UPGRADEINTERFACEVERSION is a free data retrieval call binding the contract method 0xad3cb1cc.
//
// Solidity: function UPGRADE_INTERFACE_VERSION() view returns(string)
func (_PoolFactory *PoolFactorySession) UPGRADEINTERFACEVERSION() (string, error) {
	return _PoolFactory.Contract.UPGRADEINTERFACEVERSION(&_PoolFactory.CallOpts)
}

// UPGRADEINTERFACEVERSION is a free data retrieval call binding the contract method 0xad3cb1cc.
//
// Solidity: function UPGRADE_INTERFACE_VERSION() view returns(string)
func (_PoolFactory *PoolFactoryCallerSession) UPGRADEINTERFACEVERSION() (string, error) {
	return _PoolFactory.Contract.UPGRADEINTERFACEVERSION(&_PoolFactory.CallOpts)
}

// Beacon is a free data retrieval call binding the contract method 0x59659e90.
//
// Solidity: function beacon() view returns(address)
func (_PoolFactory *PoolFactoryCaller) Beacon(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _PoolFactory.contract.Call(opts, &out, "beacon")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Beacon is a free data retrieval call binding the contract method 0x59659e90.
//
// Solidity: function beacon() view returns(address)
func (_PoolFactory *PoolFactorySession) Beacon() (common.Address, error) {
	return _PoolFactory.Contract.Beacon(&_PoolFactory.CallOpts)
}

// Beacon is a free data retrieval call binding the contract method 0x59659e90.
//
// Solidity: function beacon() view returns(address)
func (_PoolFactory *PoolFactoryCallerSession) Beacon() (common.Address, error) {
	return _PoolFactory.Contract.Beacon(&_PoolFactory.CallOpts)
}

// Created is a free data retrieval call binding the contract method 0xd42efd83.
//
// Solidity: function created(address pool) view returns(bool)
func (_PoolFactory *PoolFactoryCaller) Created(opts *bind.CallOpts, pool common.Address) (bool, error) {
	var out []interface{}
	err := _PoolFactory.contract.Call(opts, &out, "created", pool)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Created is a free data retrieval call binding the contract method 0xd42efd83.
//
// Solidity: function created(address pool) view returns(bool)
func (_PoolFactory *PoolFactorySession) Created(pool common.Address) (bool, error) {
	return _PoolFactory.Contract.Created(&_PoolFactory.CallOpts, pool)
}

// Created is a free data retrieval call binding the contract method 0xd42efd83.
//
// Solidity: function created(address pool) view returns(bool)
func (_PoolFactory *PoolFactoryCallerSession) Created(pool common.Address) (bool, error) {
	return _PoolFactory.Contract.Created(&_PoolFactory.CallOpts, pool)
}

// GetAddress is a free data retrieval call binding the contract method 0x3a98159f.
//
// Solidity: function getAddress(address _initialOwner, address _registry, bytes32[] _extensions, bytes[] _data, uint256 _salt) view returns(address)
func (_PoolFactory *PoolFactoryCaller) GetAddress(opts *bind.CallOpts, _initialOwner common.Address, _registry common.Address, _extensions [][32]byte, _data [][]byte, _salt *big.Int) (common.Address, error) {
	var out []interface{}
	err := _PoolFactory.contract.Call(opts, &out, "getAddress", _initialOwner, _registry, _extensions, _data, _salt)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetAddress is a free data retrieval call binding the contract method 0x3a98159f.
//
// Solidity: function getAddress(address _initialOwner, address _registry, bytes32[] _extensions, bytes[] _data, uint256 _salt) view returns(address)
func (_PoolFactory *PoolFactorySession) GetAddress(_initialOwner common.Address, _registry common.Address, _extensions [][32]byte, _data [][]byte, _salt *big.Int) (common.Address, error) {
	return _PoolFactory.Contract.GetAddress(&_PoolFactory.CallOpts, _initialOwner, _registry, _extensions, _data, _salt)
}

// GetAddress is a free data retrieval call binding the contract method 0x3a98159f.
//
// Solidity: function getAddress(address _initialOwner, address _registry, bytes32[] _extensions, bytes[] _data, uint256 _salt) view returns(address)
func (_PoolFactory *PoolFactoryCallerSession) GetAddress(_initialOwner common.Address, _registry common.Address, _extensions [][32]byte, _data [][]byte, _salt *big.Int) (common.Address, error) {
	return _PoolFactory.Contract.GetAddress(&_PoolFactory.CallOpts, _initialOwner, _registry, _extensions, _data, _salt)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_PoolFactory *PoolFactoryCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _PoolFactory.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_PoolFactory *PoolFactorySession) Owner() (common.Address, error) {
	return _PoolFactory.Contract.Owner(&_PoolFactory.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_PoolFactory *PoolFactoryCallerSession) Owner() (common.Address, error) {
	return _PoolFactory.Contract.Owner(&_PoolFactory.CallOpts)
}

// ProxiableUUID is a free data retrieval call binding the contract method 0x52d1902d.
//
// Solidity: function proxiableUUID() view returns(bytes32)
func (_PoolFactory *PoolFactoryCaller) ProxiableUUID(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _PoolFactory.contract.Call(opts, &out, "proxiableUUID")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// ProxiableUUID is a free data retrieval call binding the contract method 0x52d1902d.
//
// Solidity: function proxiableUUID() view returns(bytes32)
func (_PoolFactory *PoolFactorySession) ProxiableUUID() ([32]byte, error) {
	return _PoolFactory.Contract.ProxiableUUID(&_PoolFactory.CallOpts)
}

// ProxiableUUID is a free data retrieval call binding the contract method 0x52d1902d.
//
// Solidity: function proxiableUUID() view returns(bytes32)
func (_PoolFactory *PoolFactoryCallerSession) ProxiableUUID() ([32]byte, error) {
	return _PoolFactory.Contract.ProxiableUUID(&_PoolFactory.CallOpts)
}

// PoolFactoryInit is a paid mutator transaction binding the contract method 0x3368f9ec.
//
// Solidity: function __PoolFactory_init(address _owner, address _beacon) returns()
func (_PoolFactory *PoolFactoryTransactor) PoolFactoryInit(opts *bind.TransactOpts, _owner common.Address, _beacon common.Address) (*types.Transaction, error) {
	return _PoolFactory.contract.Transact(opts, "__PoolFactory_init", _owner, _beacon)
}

// PoolFactoryInit is a paid mutator transaction binding the contract method 0x3368f9ec.
//
// Solidity: function __PoolFactory_init(address _owner, address _beacon) returns()
func (_PoolFactory *PoolFactorySession) PoolFactoryInit(_owner common.Address, _beacon common.Address) (*types.Transaction, error) {
	return _PoolFactory.Contract.PoolFactoryInit(&_PoolFactory.TransactOpts, _owner, _beacon)
}

// PoolFactoryInit is a paid mutator transaction binding the contract method 0x3368f9ec.
//
// Solidity: function __PoolFactory_init(address _owner, address _beacon) returns()
func (_PoolFactory *PoolFactoryTransactorSession) PoolFactoryInit(_owner common.Address, _beacon common.Address) (*types.Transaction, error) {
	return _PoolFactory.Contract.PoolFactoryInit(&_PoolFactory.TransactOpts, _owner, _beacon)
}

// CreatePool is a paid mutator transaction binding the contract method 0xaf3ffc4d.
//
// Solidity: function createPool(address _initialOwner, address _registry, bytes32[] _extensions, bytes[] _data, uint256 _salt) returns()
func (_PoolFactory *PoolFactoryTransactor) CreatePool(opts *bind.TransactOpts, _initialOwner common.Address, _registry common.Address, _extensions [][32]byte, _data [][]byte, _salt *big.Int) (*types.Transaction, error) {
	return _PoolFactory.contract.Transact(opts, "createPool", _initialOwner, _registry, _extensions, _data, _salt)
}

// CreatePool is a paid mutator transaction binding the contract method 0xaf3ffc4d.
//
// Solidity: function createPool(address _initialOwner, address _registry, bytes32[] _extensions, bytes[] _data, uint256 _salt) returns()
func (_PoolFactory *PoolFactorySession) CreatePool(_initialOwner common.Address, _registry common.Address, _extensions [][32]byte, _data [][]byte, _salt *big.Int) (*types.Transaction, error) {
	return _PoolFactory.Contract.CreatePool(&_PoolFactory.TransactOpts, _initialOwner, _registry, _extensions, _data, _salt)
}

// CreatePool is a paid mutator transaction binding the contract method 0xaf3ffc4d.
//
// Solidity: function createPool(address _initialOwner, address _registry, bytes32[] _extensions, bytes[] _data, uint256 _salt) returns()
func (_PoolFactory *PoolFactoryTransactorSession) CreatePool(_initialOwner common.Address, _registry common.Address, _extensions [][32]byte, _data [][]byte, _salt *big.Int) (*types.Transaction, error) {
	return _PoolFactory.Contract.CreatePool(&_PoolFactory.TransactOpts, _initialOwner, _registry, _extensions, _data, _salt)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_PoolFactory *PoolFactoryTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _PoolFactory.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_PoolFactory *PoolFactorySession) RenounceOwnership() (*types.Transaction, error) {
	return _PoolFactory.Contract.RenounceOwnership(&_PoolFactory.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_PoolFactory *PoolFactoryTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _PoolFactory.Contract.RenounceOwnership(&_PoolFactory.TransactOpts)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_PoolFactory *PoolFactoryTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _PoolFactory.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_PoolFactory *PoolFactorySession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _PoolFactory.Contract.TransferOwnership(&_PoolFactory.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_PoolFactory *PoolFactoryTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _PoolFactory.Contract.TransferOwnership(&_PoolFactory.TransactOpts, newOwner)
}

// UpgradeToAndCall is a paid mutator transaction binding the contract method 0x4f1ef286.
//
// Solidity: function upgradeToAndCall(address newImplementation, bytes data) payable returns()
func (_PoolFactory *PoolFactoryTransactor) UpgradeToAndCall(opts *bind.TransactOpts, newImplementation common.Address, data []byte) (*types.Transaction, error) {
	return _PoolFactory.contract.Transact(opts, "upgradeToAndCall", newImplementation, data)
}

// UpgradeToAndCall is a paid mutator transaction binding the contract method 0x4f1ef286.
//
// Solidity: function upgradeToAndCall(address newImplementation, bytes data) payable returns()
func (_PoolFactory *PoolFactorySession) UpgradeToAndCall(newImplementation common.Address, data []byte) (*types.Transaction, error) {
	return _PoolFactory.Contract.UpgradeToAndCall(&_PoolFactory.TransactOpts, newImplementation, data)
}

// UpgradeToAndCall is a paid mutator transaction binding the contract method 0x4f1ef286.
//
// Solidity: function upgradeToAndCall(address newImplementation, bytes data) payable returns()
func (_PoolFactory *PoolFactoryTransactorSession) UpgradeToAndCall(newImplementation common.Address, data []byte) (*types.Transaction, error) {
	return _PoolFactory.Contract.UpgradeToAndCall(&_PoolFactory.TransactOpts, newImplementation, data)
}

// PoolFactoryInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the PoolFactory contract.
type PoolFactoryInitializedIterator struct {
	Event *PoolFactoryInitialized // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *PoolFactoryInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PoolFactoryInitialized)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(PoolFactoryInitialized)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *PoolFactoryInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PoolFactoryInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PoolFactoryInitialized represents a Initialized event raised by the PoolFactory contract.
type PoolFactoryInitialized struct {
	Version uint64
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0xc7f505b2f371ae2175ee4913f4499e1f2633a7b5936321eed1cdaeb6115181d2.
//
// Solidity: event Initialized(uint64 version)
func (_PoolFactory *PoolFactoryFilterer) FilterInitialized(opts *bind.FilterOpts) (*PoolFactoryInitializedIterator, error) {

	logs, sub, err := _PoolFactory.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &PoolFactoryInitializedIterator{contract: _PoolFactory.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0xc7f505b2f371ae2175ee4913f4499e1f2633a7b5936321eed1cdaeb6115181d2.
//
// Solidity: event Initialized(uint64 version)
func (_PoolFactory *PoolFactoryFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *PoolFactoryInitialized) (event.Subscription, error) {

	logs, sub, err := _PoolFactory.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PoolFactoryInitialized)
				if err := _PoolFactory.contract.UnpackLog(event, "Initialized", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseInitialized is a log parse operation binding the contract event 0xc7f505b2f371ae2175ee4913f4499e1f2633a7b5936321eed1cdaeb6115181d2.
//
// Solidity: event Initialized(uint64 version)
func (_PoolFactory *PoolFactoryFilterer) ParseInitialized(log types.Log) (*PoolFactoryInitialized, error) {
	event := new(PoolFactoryInitialized)
	if err := _PoolFactory.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PoolFactoryOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the PoolFactory contract.
type PoolFactoryOwnershipTransferredIterator struct {
	Event *PoolFactoryOwnershipTransferred // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *PoolFactoryOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PoolFactoryOwnershipTransferred)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(PoolFactoryOwnershipTransferred)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *PoolFactoryOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PoolFactoryOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PoolFactoryOwnershipTransferred represents a OwnershipTransferred event raised by the PoolFactory contract.
type PoolFactoryOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_PoolFactory *PoolFactoryFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*PoolFactoryOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _PoolFactory.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &PoolFactoryOwnershipTransferredIterator{contract: _PoolFactory.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_PoolFactory *PoolFactoryFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *PoolFactoryOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _PoolFactory.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PoolFactoryOwnershipTransferred)
				if err := _PoolFactory.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseOwnershipTransferred is a log parse operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_PoolFactory *PoolFactoryFilterer) ParseOwnershipTransferred(log types.Log) (*PoolFactoryOwnershipTransferred, error) {
	event := new(PoolFactoryOwnershipTransferred)
	if err := _PoolFactory.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PoolFactoryPoolCreatedIterator is returned from FilterPoolCreated and is used to iterate over the raw logs and unpacked data for PoolCreated events raised by the PoolFactory contract.
type PoolFactoryPoolCreatedIterator struct {
	Event *PoolFactoryPoolCreated // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *PoolFactoryPoolCreatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PoolFactoryPoolCreated)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(PoolFactoryPoolCreated)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *PoolFactoryPoolCreatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PoolFactoryPoolCreatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PoolFactoryPoolCreated represents a PoolCreated event raised by the PoolFactory contract.
type PoolFactoryPoolCreated struct {
	Pool       common.Address
	Owner      common.Address
	Registry   common.Address
	Extensions [][32]byte
	Data       [][]byte
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterPoolCreated is a free log retrieval operation binding the contract event 0x8ddf34ddd96931cdf83e601e01b38725f5ca889ae6635fe4dd7f15a65d0af67e.
//
// Solidity: event PoolCreated(address indexed pool, address indexed owner, address indexed registry, bytes32[] extensions, bytes[] data)
func (_PoolFactory *PoolFactoryFilterer) FilterPoolCreated(opts *bind.FilterOpts, pool []common.Address, owner []common.Address, registry []common.Address) (*PoolFactoryPoolCreatedIterator, error) {

	var poolRule []interface{}
	for _, poolItem := range pool {
		poolRule = append(poolRule, poolItem)
	}
	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var registryRule []interface{}
	for _, registryItem := range registry {
		registryRule = append(registryRule, registryItem)
	}

	logs, sub, err := _PoolFactory.contract.FilterLogs(opts, "PoolCreated", poolRule, ownerRule, registryRule)
	if err != nil {
		return nil, err
	}
	return &PoolFactoryPoolCreatedIterator{contract: _PoolFactory.contract, event: "PoolCreated", logs: logs, sub: sub}, nil
}

// WatchPoolCreated is a free log subscription operation binding the contract event 0x8ddf34ddd96931cdf83e601e01b38725f5ca889ae6635fe4dd7f15a65d0af67e.
//
// Solidity: event PoolCreated(address indexed pool, address indexed owner, address indexed registry, bytes32[] extensions, bytes[] data)
func (_PoolFactory *PoolFactoryFilterer) WatchPoolCreated(opts *bind.WatchOpts, sink chan<- *PoolFactoryPoolCreated, pool []common.Address, owner []common.Address, registry []common.Address) (event.Subscription, error) {

	var poolRule []interface{}
	for _, poolItem := range pool {
		poolRule = append(poolRule, poolItem)
	}
	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var registryRule []interface{}
	for _, registryItem := range registry {
		registryRule = append(registryRule, registryItem)
	}

	logs, sub, err := _PoolFactory.contract.WatchLogs(opts, "PoolCreated", poolRule, ownerRule, registryRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PoolFactoryPoolCreated)
				if err := _PoolFactory.contract.UnpackLog(event, "PoolCreated", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParsePoolCreated is a log parse operation binding the contract event 0x8ddf34ddd96931cdf83e601e01b38725f5ca889ae6635fe4dd7f15a65d0af67e.
//
// Solidity: event PoolCreated(address indexed pool, address indexed owner, address indexed registry, bytes32[] extensions, bytes[] data)
func (_PoolFactory *PoolFactoryFilterer) ParsePoolCreated(log types.Log) (*PoolFactoryPoolCreated, error) {
	event := new(PoolFactoryPoolCreated)
	if err := _PoolFactory.contract.UnpackLog(event, "PoolCreated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PoolFactoryUpgradedIterator is returned from FilterUpgraded and is used to iterate over the raw logs and unpacked data for Upgraded events raised by the PoolFactory contract.
type PoolFactoryUpgradedIterator struct {
	Event *PoolFactoryUpgraded // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *PoolFactoryUpgradedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PoolFactoryUpgraded)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(PoolFactoryUpgraded)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *PoolFactoryUpgradedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PoolFactoryUpgradedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PoolFactoryUpgraded represents a Upgraded event raised by the PoolFactory contract.
type PoolFactoryUpgraded struct {
	Implementation common.Address
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterUpgraded is a free log retrieval operation binding the contract event 0xbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b.
//
// Solidity: event Upgraded(address indexed implementation)
func (_PoolFactory *PoolFactoryFilterer) FilterUpgraded(opts *bind.FilterOpts, implementation []common.Address) (*PoolFactoryUpgradedIterator, error) {

	var implementationRule []interface{}
	for _, implementationItem := range implementation {
		implementationRule = append(implementationRule, implementationItem)
	}

	logs, sub, err := _PoolFactory.contract.FilterLogs(opts, "Upgraded", implementationRule)
	if err != nil {
		return nil, err
	}
	return &PoolFactoryUpgradedIterator{contract: _PoolFactory.contract, event: "Upgraded", logs: logs, sub: sub}, nil
}

// WatchUpgraded is a free log subscription operation binding the contract event 0xbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b.
//
// Solidity: event Upgraded(address indexed implementation)
func (_PoolFactory *PoolFactoryFilterer) WatchUpgraded(opts *bind.WatchOpts, sink chan<- *PoolFactoryUpgraded, implementation []common.Address) (event.Subscription, error) {

	var implementationRule []interface{}
	for _, implementationItem := range implementation {
		implementationRule = append(implementationRule, implementationItem)
	}

	logs, sub, err := _PoolFactory.contract.WatchLogs(opts, "Upgraded", implementationRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PoolFactoryUpgraded)
				if err := _PoolFactory.contract.UnpackLog(event, "Upgraded", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseUpgraded is a log parse operation binding the contract event 0xbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b.
//
// Solidity: event Upgraded(address indexed implementation)
func (_PoolFactory *PoolFactoryFilterer) ParseUpgraded(log types.Log) (*PoolFactoryUpgraded, error) {
	event := new(PoolFactoryUpgraded)
	if err := _PoolFactory.contract.UnpackLog(event, "Upgraded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
