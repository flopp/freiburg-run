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
});
