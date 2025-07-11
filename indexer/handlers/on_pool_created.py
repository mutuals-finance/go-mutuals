from dipdup.context import HandlerContext
from dipdup.models.evm import EvmEvent

from indexer import models as models
from indexer.types.pool_factory.evm_events.pool_created import PoolCreatedPayload


async def on_pool_created(
    ctx: HandlerContext,
    event: EvmEvent[PoolCreatedPayload],
) -> None:
    owner_address = event.payload.owner
    pool_address = event.payload.pool
    factory_address = event.payload.factory
    salt = event.payload.salt

    # Get the farm factory from the database
    pool_factory = await models.PoolFactory.get_or_none(address=factory_address)
    if not pool_factory:
        ctx.logger.warning(f'Pool factory {factory_address} not found when processing pool created event')
        return

    # Create contract and index for the new pool
    contract_name = f'pool_{pool_address[-8:]}'

    await ctx.add_contract(name=contract_name, kind='evm', address=pool_address, typename='pool')

    index_events_name = f'{contract_name}_events'
    index_transactions_name = f'{contract_name}_transactions'
    await ctx.add_index(name=index_events_name, template='pool_events', values={'contract': contract_name})
    await ctx.add_index(name=index_transactions_name, template='pool_transactions', values={'contract': contract_name})

    # Get or create the owner account model
    owner, _ = await models.Account.get_or_create(
        id=owner_address,
        defaults={
            'address': owner_address,
            'created_block': event.data.level,
            'created_transaction_hash': event.data.transaction_hash,
        },
    )

    # Create the pool
    await models.Pool.create(
        id=pool_address,
        address=pool_address,
        owner=owner,
        pool_factory=pool_factory,
        is_paused=False,
        salt=salt,
        extensions=event.payload.extensions if hasattr(event.payload, 'extensions') else None,
        initialization_data=event.payload.initialization_data
        if hasattr(event.payload, 'initialization_data')
        else None,
        created_block=event.data.level,
        created_transaction_hash=event.data.transaction_hash,
    )

    pool_factory.pool_count += 1
    await pool_factory.save()

    ctx.logger.info(
        (
            'Pool created: id={%s}, address={%s}, owner={%s}, pool_factory={%s}',
            pool_address,
            pool_address,
            owner_address,
            factory_address,
        ),
        ('added for indexing events and transactions as {%s}', contract_name),
    )
