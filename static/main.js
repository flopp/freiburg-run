document.addEventListener('DOMContentLoaded', () => {
    document.querySelectorAll('.navbar-burger').forEach(el => {
        el.addEventListener('click', () => {
            const target = el.dataset.target;
            el.classList.toggle('is-active');
            document.getElementById(target).classList.toggle('is-active'); 
        });
    });

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

        let markers = [];
        document.querySelectorAll(".event").forEach(el => {
            if (el.dataset.pending === "0") {
                return;
            }
            let geo = el.dataset.geo.trim().split(",");
            if (geo.length === 2) {
                let lat = parseFloat(geo[0]);
                let lng = parseFloat(geo[1]);
                let m = L.marker([lat, lng], {icon: greyIcon});
                markers.push(m);
                m.addTo(map);
                m.bindPopup(`${el.dataset.name}<br>${el.dataset.location}<br>NICHT BESTÄTIGT`);
            }
        });
        document.querySelectorAll(".event").forEach(el => {
            if (el.dataset.pending === "1") {
                return;
            }
            let geo = el.dataset.geo.trim().split(",");
            if (geo.length === 2) {
                let lat = parseFloat(geo[0]);
                let lng = parseFloat(geo[1]);
                let m = L.marker([lat, lng], {icon: blueIcon});
                markers.push(m);
                m.addTo(map);
                m.bindPopup(`${el.dataset.name}<br>${el.dataset.location}`);
            }
        });

        var group = new L.featureGroup(markers);
        map.fitBounds(group.getBounds(), {padding: L.point(40, 40)});
    }
});
