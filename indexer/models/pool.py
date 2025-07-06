from dipdup import fields
from dipdup.models import Model
from .account import Account
from .pool_factory import PoolFactory

class Pool(Model):
    id = fields.CharField(primary_key=True, max_length=42) # Pool address is unique

    address = fields.CharField(max_length=42) # Pool address

    owner: fields.ForeignKeyField[Account] = fields.ForeignKeyField('models.Account', related_name='pools')
    pool_factory: fields.ForeignKeyField[PoolFactory] = fields.ForeignKeyField('models.PoolFactory', related_name='pools')

    is_paused = fields.BooleanField(default=False)

    # Salt used for create2 deployment
    salt = fields.DecimalField(max_digits=78, decimal_places=0)

    extensions = fields.JSONField(null=True)  # Array of bytes32 extension IDs
    initialization_data = fields.JSONField(null=True)  # Array of bytes data

    created_block = fields.BigIntField()
    created_transaction_hash = fields.CharField(max_length=66)

    #created_at = fields.DatetimeField(auto_now_add=True)
    #updated_at = fields.DatetimeField(auto_now=True)

    class Meta:
        unique_together = (('pool_factory', 'salt'),)