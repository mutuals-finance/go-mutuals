# from dipdup.context import HandlerContext
# from dipdup.models.evm import EvmEvent
# from indexer import models as models
# from indexer.types.pool_factory.evm_events.pool_factory_initialized import PoolFactoryInitializedPayload
#
#
# async def on_pool_factory_initialized(
#     ctx: HandlerContext,
#     event: EvmEvent[PoolFactoryInitializedPayload],
# ) -> None:
#     owner_address = f'0x{event.payload.owner:x}'
#     factory_address = f'0x{event.payload.factory:x}'
#
#     # Get or create the owner account model
#     owner, _ = await models.Account.get_or_create(
#         id=owner_address,
#         defaults={
#             'address': owner_address,
#             'created_block': event.data.level,
#             'created_transaction_hash': event.data.transaction_hash,
#         },
#     )
#
#     # Get or create the pool factory from the database
#     pool_factory, created = await models.PoolFactory.get_or_create(
#         id=factory_address,
#         defaults={
#             'address': factory_address,
#             'pool_count': 0,
#             'owner': owner_address,
#             'created_block': event.data.level,
#             'created_transaction_hash': event.data.transaction_hash,
#         },
#     )
#
#     if created:
#         ctx.logger.info(
#             f'Pool factory initialized: address={factory_address}, owner={owner_address}'
#         )
#     else:
#         ctx.logger.warning(
#             f'Pool factory {factory_address} already exists, skipping initialization, owner={owner_address}'
#         )
