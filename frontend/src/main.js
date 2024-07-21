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
                    <div id="inventoryIcons"></div>
                    <div id="itemDetails"></div>
                </div>
            </div>
            <div class="side-window" id="roomToolsWindow">
                <div class="window-content">
                    <h3>Room Tools</h3>
                    <button id="captureRoomButton">Capture Room</button>
                    <div id="roomSummary"></div>
                    <div id="roomIcons"></div>
                </div>
            </div>
            <div class="side-window" id="tradeManagerWindow">
                <div class="window-content">
                    <h3>Trade Manager</h3>
                    <div id="tradeSummary"></div>
                    <div id="tradeOfferContainer"></div>
                    <div id="otherOfferContainer"></div>
                    <button id="acceptTradeButton">Accept Trade</button>
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



// Close button functionality
document.querySelector('.close').addEventListener('click', () => {
    window.go.main.App.Quit();
});

// Launch and embed Habbo
document.querySelector('#launchButton').addEventListener('click', async () => {
    const statusDiv = document.querySelector('#status');
    statusDiv.textContent = "Process started...";
    try {
        const result = await window.go.main.App.LaunchAndEmbedHabbo();
        statusDiv.textContent = result;
    } catch (error) {
        statusDiv.textContent = "Error: " + error;
    }
});

window.addEventListener('resize', () => {
    window.go.main.App.HandleResize()
        .catch(err => console.error('Error handling resize:', err));
});

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

// Event listeners for buttons
launchButton.addEventListener('click', launchAndEmbedHabbo);
scanButton.addEventListener('click', startInventoryScanning);
captureRoomButton.addEventListener('click', captureRoom);
acceptTradeButton.addEventListener('click', acceptTrade);

// Close button functionality
document.querySelectorAll('.close').forEach(closeButton => {
    closeButton.addEventListener('click', () => window.go.main.App.Quit());
});

// Event listeners for Wails runtime events
window.runtime.EventsOn("inventorySummaryUpdated", updateInventorySummary);
window.runtime.EventsOn("inventoryIconsUpdated", updateInventoryIcons);
window.runtime.EventsOn("roomUpdate", updateRoomDisplay);
window.runtime.EventsOn("tradeUpdate", updateTradeDisplay);

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
    scanButton.disabled = true;
    scanButton.textContent = "Scanning...";
    await window.go.main.App.StartInventoryScanning();
}

function updateInventorySummary(summary) {
    inventorySummary.innerHTML = `
        <h3>Inventory Summary</h3>
        <p>Total unique items: ${summary.TotalUniqueItems}</p>
        <p>Total items: ${summary.TotalItems}</p>
        <p>Total wealth: ${summary.TotalWealth.toFixed(2)} HC</p>
    `;
}

function updateInventoryIcons(groupedItems) {
    inventoryIcons.innerHTML = '';
    for (const [groupKey, items] of Object.entries(groupedItems)) {
        const item = items[0];
        const icon = createInventoryIcon(item);
        inventoryIcons.appendChild(icon);
    }
    scanButton.disabled = false;
    scanButton.textContent = "Scan Inventory";
}

function createInventoryIcon(item) {
    const icon = document.createElement('div');
    icon.className = 'inventory-icon';
    icon.style.backgroundImage = `url(${item.EnrichedItem.IconURL})`;
    icon.title = `${item.EnrichedItem.Name} (${item.Quantity})`;
    icon.onclick = () => displayItemDetails(item);
    return icon;
}

function displayItemDetails(item) {
    itemDetails.innerHTML = `
        <h3>${item.EnrichedItem.Name}</h3>
        <p>Quantity: ${item.Quantity}</p>
        <p>HC Value: ${item.EnrichedItem.HCValue.toFixed(2)}</p>
        <p>Item ID: ${item.Item.ItemId}</p>
    `;
}

async function captureRoom() {
    await window.go.main.UIManager.CaptureRoom();
}

function updateRoomDisplay(objects, items) {
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

function createRoomIcon(item) {
    const icon = document.createElement('div');
    icon.className = 'room-icon';
    icon.style.backgroundImage = `url(${item.IconURL})`;
    icon.title = item.Name;
    icon.onclick = () => window.go.main.UIManager.PickupItems([item.Id]);
    return icon;
}

async function acceptTrade() {
    await window.go.main.UIManager.AcceptTrade();
}

function updateTradeDisplay(offers) {
    tradeSummary.innerHTML = `
        <h3>Trade Summary</h3>
        <p>Your offer: ${offers.Trader.Items.length} items</p>
        <p>Their offer: ${offers.Tradee.Items.length} items</p>
    `;

    updateOfferContainer(tradeOfferContainer, offers.Trader.Items);
    updateOfferContainer(otherOfferContainer, offers.Tradee.Items);
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

window.addEventListener('resize', () => {
    window.go.main.App.HandleResize()
        .catch(err => console.error('Error handling resize:', err));
});