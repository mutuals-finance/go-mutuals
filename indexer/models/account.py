from dipdup import fields
from dipdup.models import Model


class Account(Model):
    id = fields.CharField(primary_key=True, max_length=42)  # account address

    address = fields.CharField(max_length=42)  # account address

    created_block = fields.BigIntField()
    created_transaction_hash = fields.CharField(max_length=66)

    # created_at = fields.DatetimeField(auto_now_add=True)
    # updated_at = fields.DatetimeField(auto_now=True)
