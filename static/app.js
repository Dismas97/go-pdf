let pagActual = 1;
let totalPaginas = 1;
const cantElementos = 10;

const btnAnterior = document.getElementById('btn-anterior');
const btnSiguiente = document.getElementById('btn-siguiente');
const infoPagina = document.getElementById('info-pagina');
const form = document.getElementById('form-pdf');

const divListar = document.getElementById('div-listar');

const modalFondo = document.getElementById('div-modal-fondo');
const iframe = document.getElementById('iframe-pdf');

form.addEventListener('submit', function(e) {
    e.preventDefault();
    const archivo = this.archivo.files[0];
    if (!archivo) {
        alert("Debe seleccionar un archivo.");
        return;
    }
    if (archivo.type !== "application/pdf") {
        alert("Solo se permiten archivos PDF.");
        return;
    }
    const formData = new FormData(form);
    fetch('/api/subir', {method: 'POST', body: formData})
        .then(res => res.json())
        .then(data => {
            alert(data.msj);
            cargarTabla(1);
        });
});

async function cargarTabla(pagina = 1) {
    const salto = (pagina - 1) * cantElementos;
    fetch('/api/listar?limite='+cantElementos+'&salto='+salto)
        .then(res => {return res.json();})
        .then(data => {            
            const datos = data.res;
            if(data.cant==0){
                divListar.style.display = "none";
                return false;
            }
            else {                
                divListar.style.display = "flex";
            }
            totalPaginas = Math.ceil(data.cant/cantElementos);
            renderTabla(datos);
            renderPag();
        })
}

function renderTabla(datos) {
    const tb = document.getElementById('tabla-body');
    tb.innerHTML = '';
    if (datos.length === 0) {
        return;
    }

    datos.forEach(item => {
        const fila = document.createElement('tr');        
        fila.innerHTML = '<td>'+item.id+'</td><td>'+item.nombre
            +'</td><td><a href="#" onclick="renderPdf('+item.id+'); return false;">'+item.nombre_archivo
            +'</a></td><td>'+item.creado+'</td>';
        tb.appendChild(fila);
    });
}

function renderPag() {
    infoPagina.textContent = pagActual+'/'+totalPaginas;
    btnAnterior.disabled = pagActual <= 1;
    btnSiguiente.disabled = pagActual >= totalPaginas;
}

function anterior() {
    if (pagActual > 1) {
        pagActual--;
        cargarTabla(pagActual);
    }
}

function siguiente() {
    if (pagActual < totalPaginas) {
        pagActual++;
        cargarTabla(pagActual);
    }
}

function renderPdf(id) {
    modalFondo.style.display = "flex"; 
    fetch('/api/'+id)
        .then(res => {
            console.log(res.status);
            if(!res.ok){
                return;
            }            
            const ct = res.headers.get('Content-Type');
            if(ct.includes('pdf')) {
                return res.blob().then(blob => {
                    const url = URL.createObjectURL(blob);
                    iframe.src = url;
                    iframe.style.display = 'block';
                });
            } else {
                throw new Error('Tipo no valido');
            }
        });
}


btnAnterior.addEventListener('click', anterior);
btnSiguiente.addEventListener('click', siguiente);

modalFondo.addEventListener("click", (e) => {
    if (e.target === modalFondo) {
        modalFondo.style.display = "none";
    }
});

cargarTabla(pagActual);
