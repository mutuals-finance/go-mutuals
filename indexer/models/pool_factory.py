from dipdup import fields
from dipdup.models import Model
from .account import Account

class PoolFactory(Model):
    id = fields.CharField(primary_key=True, max_length=42) # PoolFactory contract address
    address = fields.CharField(max_length=42) # PoolFactory contract address

    pool_count = fields.BigIntField(default=0)

    owner: fields.ForeignKeyField[Account] = fields.ForeignKeyField('models.Account', related_name='pool_factories')

    created_block = fields.BigIntField()
    created_transaction_hash = fields.CharField(max_length=66)

    #created_at = fields.DatetimeField(auto_now_add=True)
    #updated_at = fields.DatetimeField(auto_now=True)