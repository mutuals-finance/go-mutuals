from dipdup import fields
from dipdup.models import Model
from .token import Token
from .account import Account

class TokenBalance(Model):
    id = fields.CharField(primary_key=True, max_length=128)  # token+account address composite
    amount = fields.DecimalField(max_digits=78, decimal_places=0)

    created_block = fields.BigIntField()
    created_transaction_hash = fields.CharField(max_length=66)

    created_at = fields.DatetimeField(auto_now_add=True)
    updated_at = fields.DatetimeField(auto_now=True)

    token: fields.ForeignKeyField[Token] = fields.ForeignKeyField('models.Token', related_name='balances')
    account: fields.ForeignKeyField[Account] = fields.ForeignKeyField('models.Account', related_name='balances')

    class Meta:
        table = 'token_balance'
        unique_together = (('token', 'account'),)