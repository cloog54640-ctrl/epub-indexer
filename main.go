<!DOCTYPE html>
<html lang="fr">
<head>
  <meta charset="UTF-8">
  <title>Ma Bibliothèque EPUB</title>
  <script src="https://cdn.jsdelivr.net/npm/fuse.js@6.6.2"></script>
  <style>
    body { font-family: system-ui, -apple-system, sans-serif; background: #121212; color: #fff; margin: 20px; }
    #search { width: 100%; padding: 12px; font-size: 16px; margin-bottom: 12px; border-radius: 6px; border: 1px solid #444; background: #222; color: #fff; box-sizing: border-box; }
    #stats { font-size: 14px; color: #aaa; margin-bottom: 20px; }
    .grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(190px, 1fr)); gap: 16px; }
    .card { background: #1e1e1e; padding: 12px; border-radius: 8px; text-align: center; display: flex; flex-direction: column; justify-content: space-between; }
    .card img { max-width: 100%; height: 220px; object-fit: cover; border-radius: 4px; background-color: #2a2a2a; }
    .title { font-weight: bold; margin: 8px 0 4px; font-size: 14px; line-height: 1.2; }
    .author { font-size: 12px; color: #aaa; margin-bottom: 4px; }
    .subjects { font-size: 10px; color: #777; margin-bottom: 10px; }
    .btn-download { background: #2563eb; color: #fff; border: none; padding: 8px 12px; font-size: 12px; border-radius: 4px; cursor: pointer; font-weight: bold; text-decoration: none; display: inline-block; margin-top: auto; }
    .btn-download:hover { background: #1d4ed8; }
    #file-input-container { background: #222; padding: 15px; border-radius: 8px; border: 1px dashed #555; text-align: center; margin-bottom: 20px; }
    #file-input-container.hidden { display: none; }
  </style>
</head>
<body>

  <div id="file-input-container">
    <p style="margin-top:0;">Veuillez sélectionner votre fichier <strong>catalog.json</strong> :</p>
    <input type="file" id="json-file" accept=".json">
  </div>

  <div id="search-section" style="display:none;">
    <input type="text" id="search" placeholder="Rechercher par auteur, titre ou thème (ex: brauwn, thriller...)...">
    <div id="stats"></div>
  </div>

  <div id="results" class="grid"></div>

  <script>
    let fuse;
    let catalog = [];

    function initCatalog(data) {
      catalog = data;
      
      fuse = new Fuse(catalog, {
        keys: ['title', 'authors', 'subjects'],
        threshold: 0.4,
        distance: 100,
        ignoreLocation: true
      });

      document.getElementById('file-input-container').classList.add('hidden');
      document.getElementById('search-section').style.display = 'block';
      
      updateStats(catalog.length);
      render(catalog.slice(0, 100));
    }

    fetch('catalog.json')
      .then(res => {
        if (!res.ok) throw new Error();
        return res.json();
      })
      .then(data => initCatalog(data))
      .catch(() => {});

    document.getElementById('json-file').addEventListener('change', (e) => {
      const file = e.target.files[0];
      if (!file) return;
      const reader = new FileReader();
      reader.onload = (event) => {
        const data = JSON.parse(event.target.result);
        initCatalog(data);
      };
      reader.readAsText(file);
    });

    document.getElementById('search').addEventListener('input', (e) => {
      const query = e.target.value.trim();
      if (!query) {
        updateStats(catalog.length);
        render(catalog.slice(0, 100));
        return;
      }
      
      const results = fuse.search(query).map(r => r.item);
      updateStats(results.length);
      render(results.slice(0, 200));
    });

    function updateStats(foundCount) {
      const statsEl = document.getElementById('stats');
      const totalCount = catalog.length.toLocaleString();
      const foundFormatted = foundCount.toLocaleString();

      if (foundCount === catalog.length) {
        statsEl.textContent = `Catalogue complet : ${totalCount} livres au total.`;
      } else {
        statsEl.textContent = `${foundFormatted} livre(s) trouvé(s) / ${totalCount} livres au total.`;
      }
    }

    // Fonction de téléchargement / enregistrement vers le Bureau
    async function downloadEpub(filePath, fileName) {
      try {
        const response = await fetch(filePath);
        if (!response.ok) throw new Error("Fichier introuvable");
        const blob = await response.blob();

        // Si l'API d'enregistrement sous est disponible (Chrome/Edge/Brave)
        if ('showSaveFilePicker' in window) {
          const handle = await window.showSaveFilePicker({
            suggestedName: fileName || 'livre.epub',
            types: [{
              description: 'Fichier EPUB',
              accept: { 'application/epub+zip': ['.epub'] },
            }],
          });
          const writable = await handle.createWritable();
          await writable.write(blob);
          await writable.close();
        } else {
          // Alternative standard via téléchargement direct
          const url = window.URL.createObjectURL(blob);
          const a = document.createElement('a');
          a.href = url;
          a.download = fileName || 'livre.epub';
          document.body.appendChild(a);
          a.click();
          a.remove();
          window.URL.revokeObjectURL(url);
        }
      } catch (err) {
        if (err.name !== 'AbortError') {
          alert("Impossible de récupérer le fichier. Assurez-vous d'utiliser un serveur local ou que le chemin du fichier est accessible : " + err.message);
        }
      }
    }

    function render(books) {
      const container = document.getElementById('results');
      container.innerHTML = books.map((b, index) => {
        const safeTitle = (b.title || 'livre').replace(/[^a-z0-9]/gi, '_') + '.epub';
        return `
          <div class="card">
            <img src="${b.cover_path || 'https://via.placeholder.com/150x220?text=Pas+de+couverture'}" loading="lazy" alt="Couverture">
            <div class="title">${b.title || 'Titre inconnu'}</div>
            <div class="author">${(b.authors && b.authors.length) ? b.authors.join(', ') : 'Auteur inconnu'}</div>
            <div class="subjects">${(b.subjects && b.subjects.length) ? b.subjects.join(', ') : ''}</div>
            <button class="btn-download" onclick="downloadEpub('${encodeURIComponent(b.file_path)}', '${safeTitle}')">
               Télécharger sur le Bureau
            </button>
          </div>
        `;
      }).join('');
    }
  </script>
</body>
</html>
