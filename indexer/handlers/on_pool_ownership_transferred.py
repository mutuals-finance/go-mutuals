from dipdup.context import HandlerContext
from dipdup.models.evm import EvmEvent

from indexer import models as models
from indexer.types.pool.evm_events.ownership_transferred import OwnershipTransferredPayload


async def on_pool_ownership_transferred(
    ctx: HandlerContext,
    event: EvmEvent[OwnershipTransferredPayload],
) -> None: ...
