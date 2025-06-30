from dipdup import fields
from dipdup.models import Model

class Token(Model):
    """
    Represents a token distributed by the faucet.
    Manages token distribution settings and balances.

    Key responsibilities:
    - Tracks token availability
    - Manages claim amounts
    - Links to faucet contract
    """

    token_address = fields.TextField()  # token contract address
    owner_address = fields.TextField() # token owner address

    token_symbol = fields.TextField()
    token_name = fields.TextField()
    # Token distribution settings
    initial_amount = fields.DecimalField(max_digits=100, decimal_places=0)  # Total token amount
    amount_per_claim = fields.DecimalField(max_digits=100, decimal_places=0)  # Amount per claim
    total_claimed_amount = fields.DecimalField(
        max_digits=100, decimal_places=0, default=0
    )  # Total amount claimed by users

    created_at = fields.BigIntField()
    updated_at = fields.BigIntField()

    # Relationships
    faucet: fields.ForeignKeyField[Faucet] = fields.ForeignKeyField('models.Faucet', related_name='tokens')

    class Meta:
        unique_together = [('address', 'faucet_address')]