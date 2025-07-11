from dipdup import fields
from dipdup.models import Model


class Token(Model):
    id = fields.CharField(primary_key=True, max_length=42)  # Token contract address
    name = fields.CharField(max_length=255)
    symbol = fields.CharField(max_length=64)
    decimals = fields.DecimalField(max_digits=78, decimal_places=0)

    created_block = fields.BigIntField()
    created_transaction_hash = fields.CharField(max_length=66)

    # created_at = fields.DatetimeField(auto_now_add=True)
    # updated_at = fields.DatetimeField(auto_now=True)
