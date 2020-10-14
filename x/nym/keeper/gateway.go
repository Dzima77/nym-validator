package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/nymtech/nym/validator/nym/x/nym/types"
    "github.com/cosmos/cosmos-sdk/codec"
)

// CreateGateway creates a gateway
func (k Keeper) CreateGateway(ctx sdk.Context, gateway types.Gateway) {
	store := ctx.KVStore(k.storeKey)
	key := []byte(types.GatewayPrefix + gateway.ID)
	value := k.cdc.MustMarshalBinaryLengthPrefixed(gateway)
	store.Set(key, value)
}

// GetGateway returns the gateway information
func (k Keeper) GetGateway(ctx sdk.Context, key string) (types.Gateway, error) {
	store := ctx.KVStore(k.storeKey)
	var gateway types.Gateway
	byteKey := []byte(types.GatewayPrefix + key)
	err := k.cdc.UnmarshalBinaryLengthPrefixed(store.Get(byteKey), &gateway)
	if err != nil {
		return gateway, err
	}
	return gateway, nil
}

// SetGateway sets a gateway
func (k Keeper) SetGateway(ctx sdk.Context, gateway types.Gateway) {
	gatewayKey := gateway.ID
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshalBinaryLengthPrefixed(gateway)
	key := []byte(types.GatewayPrefix + gatewayKey)
	store.Set(key, bz)
}

// DeleteGateway deletes a gateway
func (k Keeper) DeleteGateway(ctx sdk.Context, key string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete([]byte(types.GatewayPrefix + key))
}

//
// Functions used by querier
//

func listGateway(ctx sdk.Context, k Keeper) ([]byte, error) {
	var gatewayList []types.Gateway
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, []byte(types.GatewayPrefix))
	for ; iterator.Valid(); iterator.Next() {
		var gateway types.Gateway
		k.cdc.MustUnmarshalBinaryLengthPrefixed(store.Get(iterator.Key()), &gateway)
		gatewayList = append(gatewayList, gateway)
	}
	res := codec.MustMarshalJSONIndent(k.cdc, gatewayList)
	return res, nil
}

func getGateway(ctx sdk.Context, path []string, k Keeper) (res []byte, sdkError error) {
	key := path[0]
	gateway, err := k.GetGateway(ctx, key)
	if err != nil {
		return nil, err
	}

	res, err = codec.MarshalJSONIndent(k.cdc, gateway)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}

	return res, nil
}

// Get creator of the item
func (k Keeper) GetGatewayOwner(ctx sdk.Context, key string) sdk.AccAddress {
	gateway, err := k.GetGateway(ctx, key)
	if err != nil {
		return nil
	}
	return gateway.Creator
}


// Check if the key exists in the store
func (k Keeper) GatewayExists(ctx sdk.Context, key string) bool {
	store := ctx.KVStore(k.storeKey)
	return store.Has([]byte(types.GatewayPrefix + key))
}
