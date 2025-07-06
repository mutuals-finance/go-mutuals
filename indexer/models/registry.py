from dipdup import fields
# from tortoise.fields import DatetimeField
from dipdup.models import Model
from .account import Account

class Registry(Model):
    id = fields.CharField(primary_key=True, max_length=42) # Registry contract address
    address = fields.CharField(max_length=42) # Registry contract address

    extension_count = fields.BigIntField()

    created_block = fields.BigIntField()
    created_transaction_hash = fields.CharField(max_length=66)

    #created_at = DatetimeField(auto_now_add=True)
    #updated_at = DatetimeField(auto_now=True)

    owner: fields.ForeignKeyField[Account] = fields.ForeignKeyField('models.Account', related_name='registries')
