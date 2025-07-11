from dipdup import fields
from dipdup.models import Model

from .registry import Registry


class BaseExtension(Model):
    id = fields.CharField(primary_key=True, max_length=42)  # Extension contract address
    extension_id = fields.CharField(max_length=66)  # 0x + 64 hex chars

    name = fields.CharField(max_length=255)

    registry: fields.ForeignKeyField[Registry] = fields.ForeignKeyField('models.Registry', related_name='extensions')

    is_registered = fields.BooleanField()
    registration_block = fields.BigIntField()
    registration_tx_hash = fields.CharField(max_length=66)

    registration_data = fields.TextField(null=True)
    supported_hooks = fields.JSONField(null=True)

    created_block = fields.BigIntField()
    created_transaction_hash = fields.CharField(max_length=66)

    # created_at = fields.DatetimeField(auto_now_add=True)
    # updated_at = fields.DatetimeField(auto_now=True)

    class Meta:
        unique_together = (('registry', 'extension_id'),)
