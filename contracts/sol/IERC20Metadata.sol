// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

/**
 * @title ERC-20 Token Standard, optional metadata extension
 * @dev See https://eips.ethereum.org/EIPS/eip-20
 */
interface IERC20Metadata {
    /**
    * @dev Returns the name of the token.
     */
    function name() external view returns (string memory);

    /**
     * @dev Returns the symbol of the token.
     */
    function symbol() external view returns (string memory);

    /**
     * @dev Returns the decimals places of the token.
     */
    function decimals() external view returns (uint8);
}
