from dipdup.context import HandlerContext
from dipdup.models.evm import EvmTransactionData

from indexer import models as models


async def on_all_token_transfer(
    ctx: HandlerContext,
    transaction: EvmTransactionData,
) -> None: ...
