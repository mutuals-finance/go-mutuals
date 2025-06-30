from dipdup.context import HandlerContext
from dipdup.models.starknet import StarknetEvent
from defi_space_indexer import models as models
from defi_space_indexer.types.pooling_factory.starknet_events.pool_created import PoolCreatedPayload
from defi_space_indexer.utils import get_token_info

async def on_pool_created(
    ctx: HandlerContext,
    event: StarknetEvent[PoolCreatedPayload],
) -> None:
    # Extract data from event payload
    factory_address = f'0x{event.payload.pool_factory:x}'
    lp_token_address = f'0x{event.payload.lp_token:x}'
    pool_address = f'0x{event.payload.pool:x}'
    pool_index = event.payload.pool_index
    game_session_id = event.payload.game_session_id
    block_timestamp = event.payload.block_timestamp
    penalty_duration = event.payload.penalty_duration
    withdraw_penalty = event.payload.withdraw_penalty
    multiplier = event.payload.multiplier
    penalty_receiver = f'0x{event.payload.penalty_receiver:x}'

    # Get transaction hash from event data

    lp_token_name, lp_token_symbol, _ = await get_token_info(lp_token_address)

    # Get the pool factory from the database
    pool_factory = await models.PoolFactory.get_or_none(address=factory_address)
    if not pool_factory:
        ctx.logger.warning(f'Pool factory {factory_address} not found when processing pool created event')
        return

    # Check if the pool already exists
    pool = await models.Pool.get_or_none(address=pool_address)
    if pool:
        ctx.logger.info(f'Pool {pool_address} already exists, updating details')
        pool.factory_address = factory_address
        pool.lp_token_address = lp_token_address
        pool.pool_index = pool_index
        pool.penalty_duration = penalty_duration
        pool.withdraw_penalty = withdraw_penalty
        pool.multiplier = multiplier
        pool.penalty_receiver = penalty_receiver
        pool.updated_at = block_timestamp
        await pool.save()
        return

    # Create contract and index for the new pool
    contract_name = f'pool_{pool_address[-8:]}'

    await ctx.add_contract(name=contract_name, kind='starknet', address=pool_address, typename='pooling_pool')

    index_name = f'{contract_name}_events'
    await ctx.add_index(name=index_name, template='pool_events', values={'contract': contract_name})

    # Create a new pool record
    pool = await models.Pool.create(
        address=pool_address,
        factory_address=factory_address,
        lp_token_address=lp_token_address,
        pool_index=pool_index,
        owner=pool_factory.owner,
        total_staked=0,
        multiplier=multiplier,
        penalty_duration=penalty_duration,
        withdraw_penalty=withdraw_penalty,
        penalty_receiver=penalty_receiver,
        authorized_rewarders={},
        config_history=[],
        active_rewards={},
        reward_tokens=[],
        lp_token_name=lp_token_name,
        lp_token_symbol=lp_token_symbol,
        game_session_id=int(game_session_id),
        created_at=block_timestamp,
        updated_at=block_timestamp,
        factory=pool_factory,
    )

    # Update pool count on the factory
    pool_factory.pool_count += 1
    pool_factory.updated_at = block_timestamp
    await pool_factory.save()

    ctx.logger.info(
        f'Pool created: pool={pool_address}, factory={factory_address}, '
        f'lp_token={lp_token_address}, index={pool_index}, '
        f'added for indexing as {contract_name}'
    )
