from dipdup.context import HandlerContext
from dipdup.models.evm import EvmTransaction

from indexer import models as models
from indexer.types.erc20.evm_transactions.transfer import TransferInput


async def on_token_transfer(
    ctx: HandlerContext,
    transaction: EvmTransaction[TransferInput],
) -> None:
    to_address = transaction.input.to_
    from_address = transaction.input.from_
    token_address = transaction.input.token

    created_block = transaction.data.block_number
    created_transaction_hash = transaction.data.transaction_hash

    amount = transaction.input.amount

    token, created = await models.Token.get_or_create(
        id=token_address,
        defaults={
            'address': token_address,
            'created_block': created_block,
            'created_transaction_hash': created_transaction_hash,
        },
    )

    if created:
        # If the token was newly created, store its metadata
        token.name = transaction.input.token_name
        token.symbol = transaction.input.token_symbol
        token.decimals = transaction.input.token_decimals
        await token.save()
        ctx.logger.info(f'New token created: {token_address}')

    # Get or create the to account model
    to_account, _ = await models.Account.get_or_create(
        id=to_address,
        defaults={
            'address': to_address,
            'created_block': created_block,
            'created_transaction_hash': created_transaction_hash,
        },
    )

    # Get or create the from account model
    from_account, _ = await models.Account.get_or_create(
        id=from_address,
        defaults={
            'address': from_address,
            'created_block': created_block,
            'created_transaction_hash': created_transaction_hash,
        },
    )

    # use the TokenBalance to update the balances
    from_balance, _ = await models.TokenBalance.get_or_create(
        account=from_account,
        token=transaction.input.token,
        defaults={
            'balance': 0,
            'created_block': created_block,
            'created_transaction_hash': created_transaction_hash,
        },
    )

    to_balance, to_created = await models.TokenBalance.get_or_create(
        account=to_account,
        token=token,
        defaults={
            'balance': 0,
            'created_block': created_block,
            'created_transaction_hash': created_transaction_hash,
        },
    )

    # Update to_balance
    from_balance.balance -= amount
    to_balance.balance += amount
    await from_balance.save()
    await to_balance.save()

    ctx.logger.info(f'Withdrawal: {from_address} withdrew {amount} of {token_address} to {to_address}')
