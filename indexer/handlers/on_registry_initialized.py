# from dipdup.context import HandlerContext
# from dipdup.models.evm import EvmEvent
# from indexer import models as models
# from indexer.types.registry.evm_events.registry_initialized import RegistryInitializedPayload
#
#
# async def on_registry_initialized(
#     ctx: HandlerContext,
#     event: EvmEvent[RegistryInitializedPayload],
# ) -> None:
#     owner_address = f'0x{event.payload.owner:x}'
#     registry_address = f'0x{event.data.address:x}'
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
#     registry, created = await models.Registry.get_or_create(
#         id=registry_address,
#         defaults={
#             'address': registry_address,
#             'extension_count': 0,
#             'owner': owner_address,
#             'created_block': event.data.level,
#             'created_transaction_hash': event.data.transaction_hash,
#         },
#     )
#
#     if created:
#         ctx.logger.info(
#             f'Registry initialized: address={registry_address}, owner={owner_address}'
#         )
#     else:
#         ctx.logger.warning(
#             f'Registry {registry_address} already exists, skipping initialization, owner={owner_address}'
#         )
