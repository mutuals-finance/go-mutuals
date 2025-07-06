from dipdup.context import HandlerContext
from dipdup.models.evm import EvmEvent
from indexer import models as models
from indexer.types.pool_factory.evm_events.ownership_transferred import OwnershipTransferredPayload

async def on_pool_factory_ownership_transferred(
    ctx: HandlerContext,
    event: EvmEvent[OwnershipTransferredPayload],
) -> None:
    new_owner_address = event.payload.newOwner
    factory_address = event.data.address

    # Get or create the owner account model
    new_owner, _ = await models.Account.get_or_create(
        id=new_owner_address,
        defaults={
            'address': new_owner_address,
            'created_block': event.data.level,
            'created_transaction_hash': event.data.transaction_hash,
        },
    )

    # Get or create the pool factory from the database
    pool_factory, created = await models.PoolFactory.get_or_create(
        id=factory_address,
        defaults={
            'address': factory_address,
            'pool_count': 0,
            'owner': new_owner,
            'created_block': event.data.level,
            'created_transaction_hash': event.data.transaction_hash,
        },
    )

    if created:
        ctx.logger.info(
            f'Pool factory initialized for ownership transfer: address={factory_address}, owner={new_owner_address}'
        )
    else:
        pool_factory.owner = new_owner
        await pool_factory.save()
        ctx.logger.warning(
            f'Pool factory {factory_address} already exists, updating owner, new_owner={new_owner_address}'
        )
