document.addEventListener('DOMContentLoaded', () => {
    document.querySelectorAll('.navbar-burger').forEach(el => {
        el.addEventListener('click', () => {
            const target = el.dataset.target;
            el.classList.toggle('is-active');
            document.getElementById(target).classList.toggle('is-active'); 
        });
    });

    const parseGeo = function (s) {
        const re1 = /\s*N\s*(?<lat>\d+\.\d+)\s+E\s*(?<lng>\d+\.\d+)\s*$/gm;
        const match1 = re1.exec(s);
        if (match1 !== null) {
            let lat = parseFloat(match1.groups.lat);
            let lng = parseFloat(match1.groups.lng);
            return [lat, lng];
        }

        const re2 = /\s*(?<lat>\d+\.\d+)\s*,\s*(?<lng>\d+\.\d+)\s*$/gm;
        const match2 = re2.exec(s);
        if (match2 !== null) {
            let lat = parseFloat(match2.groups.lat);
            let lng = parseFloat(match2.groups.lng);
            return [lat, lng];
        }

        return null;
    };

    if (document.querySelector("#map") !== null) {
        var map = L.map('map').setView([51.505, -0.09], 13);

        L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
            attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
        }).addTo(map);

        var freiburg = [47.996090, 7.849400];
        L.circle(freiburg, {
            color: '#3e8ed0',
            fill: false,
            weight: 1,
            radius: 25000
        }).addTo(map).bindPopup("Freiburg, 25km");
        L.circle(freiburg, {
            color: '#3e8ed0',
            fill: false,
            weight: 1,
            radius: 50000
        }).addTo(map).bindPopup("Freiburg, 50km")

        let blueOptions = {
            iconAnchor: [12, 41],
            iconRetinaUrl: "images/marker-icon-2x.png",
            iconSize: [25, 41],
            iconUrl: "images/marker-icon.png",
            popupAnchor: [1, -34],
            shadowSize: [41, 41],
            shadowUrl: "images/marker-shadow.png",
            tooltipAnchor: [16, -28],
        };
        let blueIcon = L.icon(blueOptions);

        let greyOptions = {
            iconAnchor: [12, 41],
            iconRetinaUrl: "images/marker-grey-icon-2x.png",
            iconSize: [25, 41],
            iconUrl: "images/marker-grey-icon.png",
            popupAnchor: [1, -34],
            shadowSize: [41, 41],
            shadowUrl: "images/marker-shadow.png",
            tooltipAnchor: [16, -28],
        };
        let greyIcon = L.icon(greyOptions);

        let hasPending = false;
        let markers = [];
        document.querySelectorAll(".event").forEach(el => {
            if (el.dataset.pending === "0") {
                return;
            }
            hasPending = true;
            let geo = parseGeo(el.dataset.geo);
            if (geo !== null) {
                let m = L.marker(geo, {icon: greyIcon});
                markers.push(m);
                m.addTo(map);
                m.bindPopup(`${el.dataset.name}<br>${el.dataset.location}<br>NICHT BESTÄTIGT`);
            }
        });
        document.querySelectorAll(".event").forEach(el => {
            if (el.dataset.pending === "1") {
                return;
            }
            let geo = parseGeo(el.dataset.geo);
            if (geo !== null) {
                let m = L.marker(geo, {icon: blueIcon});
                markers.push(m);
                m.addTo(map);
                m.bindPopup(`${el.dataset.name}<br>${el.dataset.location}`);
            }
            let added = Date.parse(el.dataset.added);
            const maxAge = 7 * 86400 * 1000; /* 7 days */
            if (added !== NaN && (Date.now() - added) < maxAge) {
                const dateEl = el.children[0];
                dateEl.classList.add("is-success");
                dateEl.textContent += " (neu)";
            }
        });

        const itemName = document.querySelector("body").dataset.itemtype;
        const items = [{
            label: itemName,
            type: "image",
            url: "images/marker-icon.png",
        }];
        if (hasPending) {
            items.push({
                label: `${itemName} (unbestätigt)`,
                type: "image",
                url: "images/marker-grey-icon.png"
            });
        }
        items.push(
            {
                label: "25km um Freiburg",
                type: "image",
                url: "images/circle-small.png"
            }, {
                label: "50km um Freiburg",
                type: "image",
                url: "images/circle-big.png"
            }
        );
        const legend = L.control.Legend({
            title: "Legende",
            position: "bottomleft",
            collapsed: true,
            symbolWidth: 30,
            opacity: 1,
            column: 1,
            legends: items
        }).addTo(map);

        var group = new L.featureGroup(markers);
        map.fitBounds(group.getBounds(), {padding: L.point(40, 40)});
    }

    if (document.querySelector("#parkrun-map") !== null) {
        var map = L.map('parkrun-map').setView([48.000548, 7.804842], 15);

        L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
            attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
        }).addTo(map);

        var coursePoints = [
            [48.002281,7.804621],
            [48.002315,7.804265],
            [48.002348,7.804119],
            [48.002469,7.803608],
            [48.002600,7.803354],
            [48.002763,7.803133],
            [48.002952,7.802973],
            [48.003078,7.802866],
            [48.003278,7.802675],
            [48.003514,7.802367],
            [48.003636,7.802100],
            [48.003700,7.801683],
            [48.003772,7.801243],
            [48.003669,7.801195],
            [48.003608,7.801165],
            [48.003422,7.801072],
            [48.003383,7.801054],
            [48.002737,7.800842],
            [48.002096,7.800730],
            [48.001809,7.800725],
            [48.001743,7.800731],
            [48.001446,7.800803],
            [48.001233,7.800964],
            [48.000703,7.801594],
            [48.000382,7.801761],
            [48.000190,7.801796],
            [48.000020,7.801788],
            [47.999769,7.801712],
            [47.999728,7.801698],
            [47.999630,7.801671],
            [47.999622,7.801665],
            [47.999625,7.802116],
            [47.999626,7.802235],
            [47.999641,7.804234],
            [47.999661,7.804736],
            [47.999642,7.805828],
            [47.999823,7.805947],
            [47.999965,7.806166],
            [48.000144,7.806052],
            [48.000179,7.805608],
            [48.000022,7.804995],
            [48.000022,7.804875],
            [48.000343,7.804242],
            [48.000413,7.804044],
            [48.000726,7.803563],
            [48.001049,7.803814],
            [48.001214,7.804064],
            [48.001392,7.804205],
            [48.001570,7.804292],
            [48.001989,7.803778],
            [48.002143,7.803125],
            [48.002245,7.802932],
            [48.002608,7.802530],
            [48.002764,7.802466],
            [48.002988,7.802438],
            [48.003084,7.802373],
            [48.003205,7.802232],
            [48.003373,7.801925],
            [48.003375,7.802011],
            [48.003335,7.802190],
            [48.003206,7.802389],
            [48.003185,7.802421],
            [48.002996,7.802569],
            [48.002431,7.803012],
            [48.002272,7.803253],
            [48.002266,7.803286],
            [48.002087,7.804183],
            [48.002030,7.804563],
            [48.001959,7.804773],
            [48.001307,7.805610],
            [48.000962,7.805969],
            [48.000654,7.806287],
            [48.000390,7.806800],
            [48.000205,7.807095],
            [48.000216,7.807142],
            [48.000241,7.807213],
            [48.000256,7.807237],
            [48.000490,7.807091],
            [48.000538,7.807061],
            [48.001064,7.806577],
            [48.001154,7.806495],
            [48.001370,7.806271],
            [48.001927,7.805694],
            [48.002214,7.804996],
            [48.002268,7.804758],
            [48.002281,7.804621],
            [48.002315,7.804265],
            [48.002348,7.804119],
            [48.002469,7.803608],
            [48.002600,7.803354],
            [48.002763,7.803133],
            [48.002952,7.802973],
            [48.003078,7.802866],
            [48.003278,7.802675],
            [48.003514,7.802367],
            [48.003636,7.802100],
            [48.003700,7.801683],
            [48.003772,7.801243],
            [48.003669,7.801195],
            [48.003608,7.801165],
            [48.003422,7.801072],
            [48.003383,7.801054],
            [48.002737,7.800842],
            [48.002096,7.800730],
            [48.001809,7.800725],
            [48.001743,7.800731],
            [48.001446,7.800803],
            [48.001233,7.800964],
            [48.000703,7.801594],
            [48.000382,7.801761],
            [48.000190,7.801796],
            [48.000020,7.801788],
            [47.999769,7.801712],
            [47.999728,7.801698],
            [47.999630,7.801671],
            [47.999622,7.801665],
            [47.999625,7.802116],
            [47.999626,7.802235],
            [47.999641,7.804234],
            [47.999661,7.804736],
            [47.999642,7.805828],
            [47.999823,7.805947],
            [47.999965,7.806166],
            [48.000144,7.806052],
            [48.000176,7.805643],
            [48.000179,7.805608],
            [48.000022,7.804995],
            [48.000022,7.804875],
            [48.000343,7.804242],
            [48.000413,7.804044],
            [48.000726,7.803563],
            [48.001049,7.803814],
            [48.001214,7.804064],
            [48.001392,7.804205],
            [48.001570,7.804292],
            [48.001989,7.803778],
            [48.002143,7.803125],
            [48.002245,7.802932],
            [48.002608,7.802530],
            [48.002637,7.802518],
            [48.002764,7.802466],
            [48.002988,7.802438],
            [48.003084,7.802373],
            [48.003205,7.802232],
            [48.003373,7.801925],
            [48.003375,7.802011],
            [48.003335,7.802190],
            [48.003206,7.802389],
            [48.003185,7.802421],
            [48.002996,7.802569],
            [48.003185,7.802421],
            [48.003206,7.802389],
            [48.003185,7.802421],
            [48.002996,7.802569],
            [48.002431,7.803012],
            [48.002272,7.803253],
            [48.002266,7.803286],
            [48.002087,7.804183],
            [48.002030,7.804563],
            [48.001959,7.804773],
            [48.001307,7.805610],
            [48.000962,7.805969],
            [48.000654,7.806287],
            [48.000390,7.806800],
            [48.000205,7.807095],
            [48.000216,7.807142],
            [48.000241,7.807213],
            [48.000256,7.807237],
            [48.000490,7.807091],
            [48.000538,7.807061],
            [48.001064,7.806577]            
        ];
        var course = L.polyline(coursePoints).addTo(map);

        let blueOptions = {
            iconAnchor: [12, 41],
            iconRetinaUrl: "images/marker-icon-2x.png",
            iconSize: [25, 41],
            iconUrl: "images/marker-icon.png",
            popupAnchor: [1, -34],
            shadowSize: [41, 41],
            shadowUrl: "images/marker-shadow.png",
            tooltipAnchor: [16, -28],
        };
        let blueIcon = L.icon(blueOptions);

        let greyOptions = {
            iconAnchor: [12, 41],
            iconRetinaUrl: "images/marker-grey-icon-2x.png",
            iconSize: [25, 41],
            iconUrl: "images/marker-grey-icon.png",
            popupAnchor: [1, -34],
            shadowSize: [41, 41],
            shadowUrl: "images/marker-shadow.png",
            tooltipAnchor: [16, -28],
        };
        let greyIcon = L.icon(greyOptions);
        
        let parking = L.marker([48.000993,7.808887], {icon: greyIcon});
        parking.addTo(map);
        parking.bindPopup("Parkplatz");

        let tram = L.marker([47.999420,7.810088], {icon: greyIcon});
        tram.addTo(map);
        tram.bindPopup("Straßenbahn (Linie 3, Rohrgraben)");

        let cafe = L.marker([47.997826,7.807831], {icon: greyIcon});
        cafe.addTo(map);
        cafe.bindPopup("Lio's Café");

        let meetingpoint = L.marker([48.001294,7.806489], {icon: blueIcon});
        meetingpoint.addTo(map);
        meetingpoint.bindPopup("Treffpunkt / Zielbereich");    }
});
