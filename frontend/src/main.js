import './index.css';

document.querySelector('#app').innerHTML = `
<div class="window">
    <div class="topleft"></div>
    <div class="top"></div>
    <div class="topright"></div>
    <div class="left"></div>
    <div class="content">
        <div class="main-content">
            <button id="launchButton">Launch and Embed Habbo</button>
            <div id="status"></div>
            <div id="habboContainer"></div>
        </div>
        <div class="side-windows">
            <div class="side-window" id="inventoryWindow">
                <div class="window-content">
                    <h3>Inventory</h3>
                    <button id="scanButton">Scan Inventory</button>
                    <div id="inventorySummary"></div>
                </div>
            </div>
            <div class="side-window" id="inventoryIconsWindow">
                <div class="window-content">
                    <h3>Inventory Icons</h3>
                    <div id="inventoryIcons"></div>
                </div>
            </div>
            <div class="side-window" id="itemDetailsWindow">
                <div class="window-content">
                    <h3>Item Details</h3>
                    <div id="itemDetails"></div>
                </div>
            </div>
        </div>
    </div>
    <div class="right"></div>
    <div class="bottomleft"></div>
    <div class="bottom"></div>
    <div class="bottomright"></div>
    <div class="close"></div>
</div>
`;

function log(message) {
    console.log(message);
    const logElement = document.getElementById('debugLog') || createDebugLogElement();
    logElement.innerHTML += `<p>${message}</p>`;
    logElement.scrollTop = logElement.scrollHeight;
}

function createDebugLogElement() {
    const logElement = document.createElement('div');
    logElement.id = 'debugLog';
    logElement.style.cssText = 'position: fixed; bottom: 10px; left: 10px; width: 300px; height: 200px; background: rgba(0,0,0,0.7); color: white; overflow-y: auto; padding: 10px; font-family: monospace; font-size: 12px;';
    document.body.appendChild(logElement);
    return logElement;
}

// Initialize the main window elements
const statusDiv = document.querySelector('#status');
const launchButton = document.querySelector('#launchButton');
const scanButton = document.querySelector('#scanButton');

// Initialize the inventory window elements
const inventoryWindow = document.querySelector('#inventoryWindow');
const inventorySummary = document.querySelector('#inventorySummary');
const inventoryIcons = document.querySelector('#inventoryIcons');
const itemDetails = document.querySelector('#itemDetails');

// Initialize the room tools window elements
const roomToolsWindow = document.querySelector('#roomToolsWindow');
const roomSummary = document.querySelector('#roomSummary');
const roomIcons = document.querySelector('#roomIcons');
const captureRoomButton = document.querySelector('#captureRoomButton');

// Initialize the trade manager window elements
const tradeManagerWindow = document.querySelector('#tradeManagerWindow');
const tradeSummary = document.querySelector('#tradeSummary');
const tradeOfferContainer = document.querySelector('#tradeOfferContainer');
const otherOfferContainer = document.querySelector('#otherOfferContainer');
const acceptTradeButton = document.querySelector('#acceptTradeButton');

// Close button functionality
document.querySelector('.close').addEventListener('click', () => {
    window.go.main.App.Quit();
});

// Event listeners for buttons
launchButton.addEventListener('click', launchAndEmbedHabbo);
scanButton.addEventListener('click', startInventoryScanning);
captureRoomButton?.addEventListener('click', captureRoom);
acceptTradeButton?.addEventListener('click', acceptTrade);

// Event listeners for Wails runtime events
window.runtime.EventsOn("inventorySummaryUpdated", updateInventorySummary);
window.runtime.EventsOn("inventoryDetailedSummaryUpdated", updateDetailedInventorySummary);
window.runtime.EventsOn("inventoryIconsUpdated", updateInventoryIcons);
window.runtime.EventsOn("inventoryItemIDsUpdated", updateInventoryItemIDs);
window.runtime.EventsOn("inventoryScanComplete", handleInventoryScanComplete);
window.runtime.EventsOn("inventoryScanProgress", updateScanProgress);
window.runtime.EventsOn("roomUpdate", updateRoomDisplay);
window.runtime.EventsOn("tradeUpdate", updateTradeDisplay);
window.runtime.EventsOn("inventoryItemUpdated", updateInventoryItem);

window.addEventListener('resize', () => {
    window.go.main.App.HandleResize()
        .catch(err => console.error('Error handling resize:', err));
});

async function launchAndEmbedHabbo() {
    statusDiv.textContent = "Process started...";
    try {
        const result = await window.go.main.App.LaunchAndEmbedHabbo();
        statusDiv.textContent = result;
    } catch (error) {
        statusDiv.textContent = "Error: " + error;
    }
}

async function startInventoryScanning() {
    log('Starting inventory scan...');
    scanButton.disabled = true;
    scanButton.textContent = "Scanning...";
    inventorySummary.innerHTML = '<p>Scanning inventory...</p>';
    inventoryIcons.innerHTML = '';
    itemDetails.innerHTML = '';
    try {
        await window.go.main.App.StartInventoryScanning();
        log('Inventory scan started successfully');
    } catch (error) {
        log(`Error starting inventory scan: ${error}`);
        scanButton.disabled = false;
        scanButton.textContent = "Scan Inventory";
    }
}

function updateInventorySummary(summary) {
    log(`Received inventory summary: ${JSON.stringify(summary)}`);
    inventorySummary.innerHTML = `
        <h3>Inventory Summary</h3>
        <p>Total unique items: ${summary.TotalUniqueItems}</p>
        <p>Total items: ${summary.TotalItems}</p>
        <p>Total wealth: ${summary.TotalWealth.toFixed(2)} HC</p>
    `;
}

function updateDetailedInventorySummary(detailedSummary) {
    log(`Received detailed inventory summary`);
    inventorySummary.innerHTML += `<pre>${detailedSummary}</pre>`;
}

function updateInventoryIcons(groupedItems) {
    log(`Received inventory icons: ${Object.keys(groupedItems).length} groups`);
    inventoryIcons.innerHTML = '';
    for (const [groupKey, item] of Object.entries(groupedItems)) {
        const icon = createInventoryIcon(item);
        inventoryIcons.appendChild(icon);
    }
}

function updateInventoryItemIDs(itemIDs) {
    log(`Received inventory item IDs`);
    itemDetails.innerHTML = `<pre>${itemIDs}</pre>`;
}

function createInventoryIcon(item) {
    const icon = document.createElement('div');
    icon.className = 'inventory-icon';
    icon.style.backgroundImage = `url(${item.EnrichedItem.IconURL})`;
    icon.title = `${item.EnrichedItem.Name} (${item.Quantity})`;
    icon.onclick = () => displayItemDetails(item);
    
    const quantityLabel = document.createElement('span');
    quantityLabel.className = 'quantity-label';
    quantityLabel.textContent = item.Quantity;
    icon.appendChild(quantityLabel);
    
    return icon;
}

function displayItemDetails(item) {
    let itemIDs = item.Items.map(i => i.ItemId).join('\n');
    itemDetails.innerHTML = `
        <h3>${item.EnrichedItem.Name}</h3>
        <p>Quantity: ${item.Quantity}</p>
        <p>HC Value: ${(item.EnrichedItem.HCValue * item.Quantity).toFixed(2)}</p>
        <p>Item IDs:</p>
        <pre>${itemIDs}</pre>
    `;
}

function handleInventoryScanComplete() {
    log('Received inventoryScanComplete event');
    scanButton.disabled = false;
    scanButton.textContent = "Scan Inventory";
}

function updateScanProgress(itemCount) {
    log(`Scan progress: ${itemCount} items scanned`);
}

async function captureRoom() {
    await window.go.main.App.CaptureRoom();
}

function updateRoomDisplay(objects, items) {
    if (roomSummary && roomIcons) {
        roomSummary.innerHTML = `
            <h3>Room Summary</h3>
            <p>Floor items: ${objects.length}</p>
            <p>Wall items: ${items.length}</p>
        `;

        roomIcons.innerHTML = '';
        [...objects, ...items].forEach(item => {
            const icon = createRoomIcon(item);
            roomIcons.appendChild(icon);
        });
    }
}

function createRoomIcon(item) {
    const icon = document.createElement('div');
    icon.className = 'room-icon';
    icon.style.backgroundImage = `url(${item.IconURL})`;
    icon.title = item.Name;
    icon.onclick = () => window.go.main.App.PickupItems([item.Id]);
    return icon;
}

async function acceptTrade() {
    await window.go.main.App.AcceptTrade();
}

function updateTradeDisplay(offers) {
    if (tradeSummary && tradeOfferContainer && otherOfferContainer) {
        tradeSummary.innerHTML = `
            <h3>Trade Summary</h3>
            <p>Your offer: ${offers.Trader.Items.length} items</p>
            <p>Their offer: ${offers.Tradee.Items.length} items</p>
        `;

        updateOfferContainer(tradeOfferContainer, offers.Trader.Items);
        updateOfferContainer(otherOfferContainer, offers.Tradee.Items);
    }
}

function updateOfferContainer(container, items) {
    container.innerHTML = '';
    items.forEach(item => {
        const icon = createTradeIcon(item);
        container.appendChild(icon);
    });
}

function createTradeIcon(item) {
    const icon = document.createElement('div');
    icon.className = 'trade-icon';
    icon.style.backgroundImage = `url(${item.IconURL})`;
    icon.title = item.Name;
    return icon;
}

function updateInventoryItem(data) {
    updateInventorySummary(data.summary);
    updateInventoryIcons(data.groupedItems);
    highlightUpdatedItem(data.updatedItem, data.isAddition);
}

function highlightUpdatedItem(item, isAddition) {
    const icons = document.querySelectorAll('.inventory-icon');
    for (const icon of icons) {
        if (icon.title.includes(item.EnrichedItem.Name)) {
            icon.classList.add(isAddition ? 'highlight-add' : 'highlight-remove');
            setTimeout(() => {
                icon.classList.remove('highlight-add', 'highlight-remove');
            }, 2000);
            break;
        }
    }
}

// Add this CSS to your stylesheet
const style = document.createElement('style');
style.textContent = `
.highlight-add {
    animation: pulse-green 2s;
}

.highlight-remove {
    animation: pulse-red 2s;
}

@keyframes pulse-green {
    0% { box-shadow: 0 0 0 0 rgba(0, 255, 0, 0.7); }
    70% { box-shadow: 0 0 0 10px rgba(0, 255, 0, 0); }
    100% { box-shadow: 0 0 0 0 rgba(0, 255, 0, 0); }
}

@keyframes pulse-red {
    0% { box-shadow: 0 0 0 0 rgba(255, 0, 0, 0.7); }
    70% { box-shadow: 0 0 0 10px rgba(255, 0, 0, 0); }
    100% { box-shadow: 0 0 0 0 rgba(255, 0, 0, 0); }
}
`;
document.head.appendChild(style);

// Add event listeners for item placement and pickup
document.querySelectorAll('.place-item').forEach(button => {
    button.addEventListener('click', () => {
        const itemId = button.getAttribute('data-item-id');
        const x = parseInt(button.getAttribute('data-x'));
        const y = parseInt(button.getAttribute('data-y'));
        window.go.main.App.PlaceItem(itemId, x, y);
    });
});

document.querySelectorAll('.pickup-item').forEach(button => {
    button.addEventListener('click', () => {
        const itemId = button.getAttribute('data-item-id');
        window.go.main.App.PickupItems([itemId]);
    });
});
