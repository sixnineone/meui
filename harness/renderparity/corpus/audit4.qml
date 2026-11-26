// Component audit 4: DataTable feature matrix.
Window {
    width: 940
    height: 500

    VStack {
        spacing: 10
        margins: 12

        Label { text: "DataTable — typed columns, cell colors, row numbers:"; color: "#AAAAAA" }

        DataTable {
            width: 900
            height: 300
            columns: [
                { label: "Symbol", type: "text", width: 120 },
                { label: "Bid", type: "number", width: 120 },
                { label: "Ask", type: "number", width: 120 },
                { label: "Change", type: "percent", width: 110 },
                { label: "Volume", type: "number" }
            ]
            showRowNumbers: true
            alternatingRows: true
            showGrid: true
            rows: [
                [ { text: "EURUSD", color: "#FFFFFF" }, { text: "1.08421" }, { text: "1.08435" }, { text: "+0.12%", color: "#4CAF50" }, { text: "152300" } ],
                [ { text: "GBPUSD", color: "#FFFFFF" }, { text: "1.26710" }, { text: "1.26730" }, { text: "-0.08%", color: "#F44336" }, { text: "98400" } ],
                [ { text: "USDJPY", color: "#FFFFFF" }, { text: "151.302" }, { text: "151.325", bgColor: "#3A5A3A" }, { text: "+0.35%", color: "#4CAF50" }, { text: "210100" } ],
                [ { text: "AUDUSD", color: "#FFFFFF" }, { text: "0.65310" }, { text: "0.65330" }, { text: "-0.21%", color: "#F44336" }, { text: "45200" } ],
                [ { text: "USDCAD", color: "#FFFFFF" }, { text: "1.35840" }, { text: "1.35860" }, { text: "+0.03%", color: "#4CAF50" }, { text: "30100" } ],
                [ { text: "USDCHF", color: "#FFFFFF" }, { text: "0.90210" }, { text: "0.90240" }, { text: "-0.15%", color: "#F44336" }, { text: "27800" } ]
            ]
        }

        Label { text: "Click a header to sort; drag a header edge to resize; drag a header to reorder."; color: "#666666"; height: 18 }
    }
}
