package ui

import (
	"context"
	"fmt"
	"sync"

	"github.com/bolognesandwiches/G-itemViewer/common"
	"github.com/bolognesandwiches/G-itemViewer/trading"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	g "xabbo.b7c.io/goearth"
	"xabbo.b7c.io/goearth/shockwave/inventory"
	"xabbo.b7c.io/goearth/shockwave/out"
	"xabbo.b7c.io/goearth/shockwave/profile"
	"xabbo.b7c.io/goearth/shockwave/room"
	"xabbo.b7c.io/goearth/shockwave/trade"
)

type UIManager struct {
	ctx              context.Context
	ext              *g.Ext
	inventoryManager *inventory.Manager
	roomManager      *room.Manager
	tradeManager     *trading.Manager
	profileManager   *profile.Manager
	unifiedInventory *UnifiedInventory
	mu               sync.Mutex
}

func NewUIManager(ctx context.Context, ext *g.Ext, inventoryManager *inventory.Manager, roomManager *room.Manager, profileManager *profile.Manager, tradeManager *trading.Manager, startInventoryScanning func()) *UIManager {
	return &UIManager{
		ctx:              ctx,
		ext:              ext,
		inventoryManager: inventoryManager,
		roomManager:      roomManager,
		profileManager:   profileManager,
		tradeManager:     tradeManager,
		unifiedInventory: NewUnifiedInventory(),
	}
}

type UnifiedItem struct {
	Item         inventory.Item
	EnrichedItem common.EnrichedInventoryItem
	Quantity     int
	InTrade      bool
}

type UnifiedInventory struct {
	Items   map[int]UnifiedItem
	Summary InventorySummary
	mu      sync.RWMutex
}

type InventorySummaryItem struct {
	Quantity int
	HCValue  float64
}

type InventorySummary struct {
	TotalUniqueItems int
	TotalItems       int
	TotalWealth      float64
	Items            map[string]InventorySummaryItem
}

func NewUnifiedInventory() *UnifiedInventory {
	return &UnifiedInventory{
		Items: make(map[int]UnifiedItem),
		Summary: InventorySummary{
			Items: make(map[string]InventorySummaryItem),
		},
	}
}

func (ui *UnifiedInventory) AddItem(item inventory.Item) {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	enrichedItem := common.EnrichInventoryItem(item)
	unifiedItem, exists := ui.Items[item.ItemId]
	if !exists {
		unifiedItem = UnifiedItem{
			Item:         item,
			EnrichedItem: enrichedItem,
			Quantity:     1,
			InTrade:      false,
		}
		ui.Summary.TotalUniqueItems++
	} else {
		unifiedItem.Quantity++
	}
	ui.Items[item.ItemId] = unifiedItem

	ui.Summary.TotalItems++
	ui.Summary.TotalWealth += enrichedItem.HCValue

	summaryItem := ui.Summary.Items[enrichedItem.Name]
	summaryItem.Quantity++
	summaryItem.HCValue = enrichedItem.HCValue
	ui.Summary.Items[enrichedItem.Name] = summaryItem
}

func (ui *UnifiedInventory) GetSummary() InventorySummary {
	ui.mu.RLock()
	defer ui.mu.RUnlock()
	return ui.Summary
}

func (ui *UnifiedInventory) GetGroupedItems() map[string][]UnifiedItem {
	ui.mu.RLock()
	defer ui.mu.RUnlock()

	grouped := make(map[string][]UnifiedItem)
	for _, item := range ui.Items {
		key := item.EnrichedItem.GroupKey
		grouped[key] = append(grouped[key], item)
	}
	return grouped
}

func (u *UIManager) ScanInventory() {
	u.unifiedInventory = NewUnifiedInventory()
	u.inventoryManager.Update()
}

func (u *UIManager) HandleInventoryUpdate() {
	items := u.inventoryManager.Items()
	for _, item := range items {
		u.unifiedInventory.AddItem(item)
	}
	u.RefreshInventoryDisplay()

	// Continue scanning if there are more items
	if len(items) > 0 {
		u.ext.Send(out.GETSTRIP, []byte("next"))
	} else {
		// Scanning is complete
		runtime.EventsEmit(u.ctx, "inventoryScanComplete")
	}
}

func (u *UIManager) RefreshInventoryDisplay() {
	summary := u.unifiedInventory.GetSummary()
	groupedItems := u.unifiedInventory.GetGroupedItems()

	runtime.EventsEmit(u.ctx, "inventorySummaryUpdated", summary)
	runtime.EventsEmit(u.ctx, "inventoryIconsUpdated", groupedItems)
}

func (u *UIManager) HandleRoomUpdate(args room.ObjectsArgs) {
	objects := u.roomManager.Objects
	items := u.roomManager.Items

	enrichedObjects := make([]common.EnrichedRoomObject, 0, len(objects))
	for _, obj := range objects {
		enrichedObjects = append(enrichedObjects, common.EnrichRoomObject(obj))
	}

	enrichedItems := make([]common.EnrichedRoomItem, 0, len(items))
	for _, item := range items {
		enrichedItems = append(enrichedItems, common.EnrichRoomItem(item))
	}

	runtime.EventsEmit(u.ctx, "roomUpdate", enrichedObjects, enrichedItems)
}

func (u *UIManager) HandleTradeUpdate(args trade.Args) {
	offers := trading.Offers{
		Trader: args.Offers[0],
		Tradee: args.Offers[1],
	}
	runtime.EventsEmit(u.ctx, "tradeUpdate", offers)
}

func (u *UIManager) CaptureRoom() {
	// Implement room capture logic
	runtime.EventsEmit(u.ctx, "roomCaptured")
}

func (u *UIManager) PickupItems(itemIds []int) {
	for _, id := range itemIds {
		u.ext.Send(out.ADDSTRIPITEM, []byte(fmt.Sprintf("new stuff %d", id)))
	}
	runtime.EventsEmit(u.ctx, "itemsPickedUp", itemIds)
}

func (u *UIManager) AcceptTrade() {
	u.tradeManager.Accept()
	runtime.EventsEmit(u.ctx, "tradeAccepted")
}

func (u *UIManager) GetRoomSummary() string {
	objects := u.roomManager.Objects
	items := u.roomManager.Items
	return common.GetRoomSummary(objects, items)
}

func (u *UIManager) OfferItem(itemId int) {
	u.tradeManager.Offer(itemId)
}
