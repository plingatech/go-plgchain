// SPDX-License-Identifier: MIT
pragma solidity 0.8.17;

import "@openzeppelin/contracts/token/ERC20/extensions/IERC20Metadata.sol";

interface IRootERC20Predicate {
    struct ERC20BridgeEvent {
        address rootToken;
        address childToken;
        address sender;
        address receiver;
    }

    event ERC20Deposit(
        address indexed rootToken,
        address indexed childToken,
        address depositor,
        address indexed receiver,
        uint256 amount
    );
    event ERC20Withdraw(
        address indexed rootToken,
        address indexed childToken,
        address withdrawer,
        address indexed receiver,
        uint256 amount
    );
    event TokenMapped(address indexed rootToken, address indexed childToken);

    /**
     * @notice Function to deposit tokens from the depositor to themselves on the child chain
     * @param rootToken Address of the root token being deposited
     * @param amount Amount to deposit
     */
    function deposit(IERC20Metadata rootToken, uint256 amount) external;

    /**
     * @notice Function to deposit tokens from the depositor to another address on the child chain
     * @param rootToken Address of the root token being deposited
     * @param amount Amount to deposit
     */
    function depositTo(IERC20Metadata rootToken, address receiver, uint256 amount) external;

    /**
     * @notice Function to be used for token mapping
     * @param rootToken Address of the root token to map
     * @dev Called internally on deposit if token is not mapped already
     */
    function mapToken(IERC20Metadata rootToken) external;
}
