(function() {
  var container = document.querySelector('.retoots-container');
  var status = document.querySelector('.retoots-container a').getAttribute('href');
  console.log(status);
  fetch('https://gograz.org/retoots/api/v1/interactions?status=' + encodeURIComponent(status)).then(resp => {
    return resp.json();
  }).then(data => {
    if (!data.descendants.length) {
      var emptyMessage = document.createElement('p');
      emptyMessage.className = 'retoots__empty';
      emptyMessage.innerHTML = 'Be the first to leave a comment!';
      container.appendChild(emptyMessage);
    } else {
      var commentList = document.createElement('div');
      commentList.className = 'retoots__listing';
      data.descendants.forEach(comment => {
        var cc = document.createElement('div');
        cc.className = 'comment';
        var av = document.createElement('div');
        av.className = 'comment__avatar';
        var avl = document.createElement('a');
        avl.setAttribute('href', comment.account.url);
        var avi = document.createElement('img');
        avi.setAttribute('src', comment.account.avatar);
        avl.appendChild(avi);
        av.appendChild(avl);
        cc.appendChild(av);
        var content = document.createElement('div');
        content.innerHTML = comment.content;
        content.className = 'comment__content';
        cc.appendChild(content);
        var cl = document.createElement('a');
        cl.className = 'comment__date';
        cl.innerHTML = comment.created_at;
        cl.setAttribute('href', comment.url);
        content.appendChild(cl);
        commentList.appendChild(cc);
      });
      container.appendChild(commentList);
    }
  });
})();
