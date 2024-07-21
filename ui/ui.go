package ui

import (
	"context"
	"sync"

	"github.com/bolognesandwiches/G-itemViewer/common"
	"github.com/bolognesandwiches/G-itemViewer/trading"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	g "xabbo.b7c.io/goearth"
	"xabbo.b7c.io/goearth/shockwave/inventory"
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

type UnifiedItem struct {
	Items        []inventory.Item
	EnrichedItem common.EnrichedInventoryItem
	Quantity     int
	InTrade      bool
}

type UnifiedInventory struct {
	Items   map[string]UnifiedItem
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

func NewUnifiedInventory() *UnifiedInventory {
	return &UnifiedInventory{
		Items: make(map[string]UnifiedItem),
		Summary: InventorySummary{
			Items: make(map[string]InventorySummaryItem),
		},
	}
}

func (ui *UnifiedInventory) AddItem(item inventory.Item) {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	enrichedItem := common.EnrichInventoryItem(item)
	groupKey := enrichedItem.GroupKey
	unifiedItem, exists := ui.Items[groupKey]
	if !exists {
		unifiedItem = UnifiedItem{
			Items:        []inventory.Item{item},
			EnrichedItem: enrichedItem,
			Quantity:     1,
			InTrade:      false,
		}
		ui.Summary.TotalUniqueItems++
	} else {
		unifiedItem.Items = append(unifiedItem.Items, item)
		unifiedItem.Quantity++
	}
	ui.Items[groupKey] = unifiedItem

	ui.Summary.TotalItems++
	ui.Summary.TotalWealth += enrichedItem.HCValue

	summaryItem := ui.Summary.Items[enrichedItem.Name]
	summaryItem.Quantity++
	summaryItem.HCValue += enrichedItem.HCValue
	ui.Summary.Items[enrichedItem.Name] = summaryItem
}

func (ui *UnifiedInventory) RemoveItem(itemId int) {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	for groupKey, unifiedItem := range ui.Items {
		for i, item := range unifiedItem.Items {
			if item.ItemId == itemId {
				unifiedItem.Items = append(unifiedItem.Items[:i], unifiedItem.Items[i+1:]...)
				unifiedItem.Quantity--

				ui.Summary.TotalItems--
				ui.Summary.TotalWealth -= unifiedItem.EnrichedItem.HCValue

				summaryItem := ui.Summary.Items[unifiedItem.EnrichedItem.Name]
				summaryItem.Quantity--
				summaryItem.HCValue -= unifiedItem.EnrichedItem.HCValue

				if unifiedItem.Quantity == 0 {
					delete(ui.Items, groupKey)
					ui.Summary.TotalUniqueItems--
					delete(ui.Summary.Items, unifiedItem.EnrichedItem.Name)
				} else {
					ui.Items[groupKey] = unifiedItem
					ui.Summary.Items[unifiedItem.EnrichedItem.Name] = summaryItem
				}

				return
			}
		}
	}
}

func (ui *UnifiedInventory) UpdateItemTradeStatus(itemId int, inTrade bool) {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	for groupKey, unifiedItem := range ui.Items {
		for _, item := range unifiedItem.Items {
			if item.ItemId == itemId {
				unifiedItem.InTrade = inTrade // Update the UnifiedItem's InTrade status
				ui.Items[groupKey] = unifiedItem
				return
			}
		}
	}
}

func (ui *UnifiedInventory) GetGroupedItems() map[string]UnifiedItem {
	ui.mu.RLock()
	defer ui.mu.RUnlock()
	return ui.Items
}

func (ui *UnifiedInventory) GetSummary() InventorySummary {
	ui.mu.RLock()
	defer ui.mu.RUnlock()
	return ui.Summary
}

func (ui *UnifiedInventory) ItemExists(itemId int) bool {
	ui.mu.RLock()
	defer ui.mu.RUnlock()
	for _, unifiedItem := range ui.Items {
		for _, item := range unifiedItem.Items {
			if item.ItemId == itemId {
				return true
			}
		}
	}
	return false
}

func (m *UIManager) HandleInventoryUpdate() {
	m.mu.Lock()
	defer m.mu.Unlock()

	items := m.inventoryManager.Items()
	m.unifiedInventory = NewUnifiedInventory()
	for _, item := range items {
		m.unifiedInventory.AddItem(item)
	}

	m.RefreshInventoryDisplay()
	runtime.EventsEmit(m.ctx, "inventoryScanComplete")
}
func (m *UIManager) RefreshInventorySummaryDisplay() {
	summary := m.unifiedInventory.GetSummary()
	runtime.EventsEmit(m.ctx, "inventorySummaryUpdated", summary)
}

func (m *UIManager) RefreshInventoryIcons() {
	groupedItems := m.unifiedInventory.GetGroupedItems()
	runtime.EventsEmit(m.ctx, "inventoryIconsUpdated", groupedItems)
}

func (m *UIManager) HandleItemAddition(item inventory.Item) {
	m.unifiedInventory.AddItem(item)
	m.RefreshInventoryDisplay()
}

func (m *UIManager) HandleItemRemoval(itemId int) {
	m.unifiedInventory.RemoveItem(itemId)
	m.RefreshInventoryDisplay()
}

func (m *UIManager) RefreshInventoryDisplay() {
	m.RefreshInventorySummaryDisplay()
	m.RefreshInventoryIcons()
}

func (m *UIManager) HandleTradeUpdated(args trade.Args) {
	offers := trading.Offers{
		Trader: args.Offers[0],
		Tradee: args.Offers[1],
	}
	runtime.EventsEmit(m.ctx, "tradeUpdate", offers)
}
func (m *UIManager) HandleTradeAccepted(args trade.AcceptArgs) {
	runtime.EventsEmit(m.ctx, "tradeAccepted", args)
}

func (m *UIManager) HandleTradeCompleted(args trade.Args) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, item := range args.Offers[0].Items {
		m.unifiedInventory.RemoveItem(item.ItemId)
	}
	for _, item := range args.Offers[1].Items {
		m.unifiedInventory.AddItem(item)
	}

	m.RefreshInventorySummaryDisplay()
	m.RefreshInventoryIcons()
	runtime.EventsEmit(m.ctx, "tradeCompleted", args)
}
func (m *UIManager) HandleTradeClosed(args trade.Args) {
	// Reset trade status for all items
	for _, unifiedItem := range m.unifiedInventory.Items {
		for _, item := range unifiedItem.Items {
			m.unifiedInventory.UpdateItemTradeStatus(item.ItemId, false)
		}
	}
	runtime.EventsEmit(m.ctx, "tradeClosed", args)
}

func (m *UIManager) AddItemToRoom(item room.Object) {
	enrichedObject := common.EnrichRoomObject(item)
	runtime.EventsEmit(m.ctx, "roomItemAdded", enrichedObject)
}

func (m *UIManager) RemoveItemFromRoom(itemId int) {
	runtime.EventsEmit(m.ctx, "roomItemRemoved", itemId)
}

func (m *UIManager) UpdateRoomDisplay(objects map[int]room.Object, items map[int]room.Item) {
	enrichedObjects := make([]common.EnrichedRoomObject, 0, len(objects))
	for _, obj := range objects {
		enrichedObjects = append(enrichedObjects, common.EnrichRoomObject(obj))
	}

	enrichedItems := make([]common.EnrichedRoomItem, 0, len(items))
	for _, item := range items {
		enrichedItems = append(enrichedItems, common.EnrichRoomItem(item))
	}

	runtime.EventsEmit(m.ctx, "roomUpdate", enrichedObjects, enrichedItems)
}

func (m *UIManager) CaptureRoom() {
	objects := m.roomManager.Objects
	items := m.roomManager.Items
	summary := common.GetRoomSummary(objects, items)
	runtime.EventsEmit(m.ctx, "roomCaptured", summary)
}

func (m *UIManager) AcceptTrade() {
	m.tradeManager.Accept()
	runtime.EventsEmit(m.ctx, "tradeAccepted")
}

func (m *UIManager) OfferItem(itemId int) {
	m.tradeManager.Offer(itemId)
	m.unifiedInventory.UpdateItemTradeStatus(itemId, true)
	runtime.EventsEmit(m.ctx, "itemOffered", itemId)
}

func (m *UIManager) UpdateInventoryDisplay(items map[int]inventory.Item) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.unifiedInventory = NewUnifiedInventory()
	for _, item := range items {
		m.unifiedInventory.AddItem(item)
	}

	m.RefreshInventorySummaryDisplay()
	m.RefreshInventoryIcons()
}

func (m *UIManager) UpdateInventoryItem(item inventory.Item, isAddition bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if isAddition {
		m.unifiedInventory.AddItem(item)
	} else {
		m.unifiedInventory.RemoveItem(item.ItemId)
	}

	// Get updated summary and grouped items
	summary := m.unifiedInventory.GetSummary()
	groupedItems := m.unifiedInventory.GetGroupedItems()

	// Emit the update event
	runtime.EventsEmit(m.ctx, "inventoryItemUpdated", map[string]interface{}{
		"summary":      summary,
		"groupedItems": groupedItems,
		"updatedItem":  item,
		"isAddition":   isAddition,
	})

	runtime.LogInfof(m.ctx, "Inventory item updated: %+v, isAddition: %v", item, isAddition)
}

func (m *UIManager) FindItemById(itemId int) (inventory.Item, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, unifiedItem := range m.unifiedInventory.Items {
		for _, item := range unifiedItem.Items {
			if item.ItemId == itemId {
				return item, true
			}
		}
	}
	return inventory.Item{}, false
}
