from dipdup.context import HandlerContext
from dipdup.models.evm import EvmEvent

from indexer.models import Account
from indexer.models import Registry
from indexer.types.registry.evm_events.ownership_transferred import OwnershipTransferredPayload


async def on_registry_ownership_transferred(
    ctx: HandlerContext,
    event: EvmEvent[OwnershipTransferredPayload],
) -> None:
    new_owner_address = event.payload.newOwner
    registry_address = event.data.address

    # Get or create the owner account model
    new_owner, _ = await Account.get_or_create(
        id=new_owner_address,
        defaults={
            'address': new_owner_address,
            'created_block': event.data.level,
            'created_transaction_hash': event.data.transaction_hash,
        },
    )

    # Get or create the registry from the database
    registry, created = await Registry.get_or_create(
        id=registry_address,
        defaults={
            'address': registry_address,
            'extension_count': 0,
            'created_block': event.data.level,
            'created_transaction_hash': event.data.transaction_hash,
            'owner': new_owner,
        },
    )

    if created:
        ctx.logger.info(f'Registry initialized: address={registry_address}, owner={new_owner_address}')
    else:
        registry.owner = new_owner
        await registry.save()
        ctx.logger.warning(f'Registry {registry_address} already exists, updated owner, owner={new_owner_address}')
