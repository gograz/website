(function() {
    var container = document.getElementById('posRSVPs');
    if (!container || typeof fetch === 'undefined') {
        return;
    }
    var meetupId = container.getAttribute("data-meetup-id");
    if (!meetupId) {
        return;
    }
    fetch('https://api.gograz.org/meetup/' + meetupId + '/rsvps').then(resp => {
        return resp.json();
    }).then(data => {
        data.yes.forEach((member, idx) => {
            var link = document.createElement('a');
            link.setAttribute('class', 'member');
            link.setAttribute('href', 'https://www.meetup.com/Graz-Open-Source-Meetup/members/' + member.id);
            link.setAttribute('title', member.name);
            var img = document.createElement('img');
            if (!member.thumbLink) {
                member.thumbLink = '/images/gopher.png';
            }
            img.setAttribute('src', member.thumbLink);
            img.setAttribute('alt', member.name);
            link.appendChild(img);
            container.appendChild(link);
        });
    });
})();
