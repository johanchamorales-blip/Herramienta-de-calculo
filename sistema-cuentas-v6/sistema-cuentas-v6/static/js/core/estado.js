import { api } from './api.js';

// Estado compartido: quién está conectado, los catálogos y la cuenta sobre la
// que se está trabajando. Las vistas se suscriben en vez de recargar todo.
class Estado extends EventTarget {
  constructor() {
    super();
    this.usuario = null;
    this.permisos = [];
    this.cooperativas = [];
    this.bancos = [];
    this.cuentas = [];
    this.cooperativaID = '';
    this.cuentaID = '';
  }

  puede(permiso) {
    return this.permisos.includes(permiso);
  }

  get cuenta() {
    return this.cuentas.find((c) => String(c.id) === String(this.cuentaID)) || null;
  }

  get cooperativa() {
    const cuenta = this.cuenta;
    const id = cuenta ? cuenta.cooperativa_id : this.cooperativaID;
    return this.cooperativas.find((c) => String(c.id) === String(id)) || null;
  }

  banco(id) {
    return this.bancos.find((b) => String(b.id) === String(id)) || null;
  }

  cuentasDeCooperativa(cooperativaID = this.cooperativaID) {
    if (!cooperativaID) return this.cuentas;
    return this.cuentas.filter((c) => String(c.cooperativa_id) === String(cooperativaID));
  }

  etiquetaCuenta(cuenta) {
    if (!cuenta) return '';
    const banco = this.banco(cuenta.banco_id);
    return `${cuenta.numero} · ${cuenta.nombre}${banco ? ` · ${banco.nombre}` : ''}`;
  }

  async cargarCatalogos() {
    const [cooperativas, bancos, cuentas] = await Promise.all([
      api.obtener('/api/cooperativas'),
      api.obtener('/api/bancos'),
      api.obtener('/api/cuentas'),
    ]);
    this.cooperativas = cooperativas;
    this.bancos = bancos;
    this.cuentas = cuentas;

    // Si la cuenta elegida desapareció, se elige la primera disponible.
    if (!this.cuenta) {
      const candidatas = this.cuentasDeCooperativa();
      this.cuentaID = candidatas.length ? String(candidatas[0].id) : '';
    }
    if (!this.cooperativaID && this.cuenta) {
      this.cooperativaID = String(this.cuenta.cooperativa_id);
    }
    this.anunciar('catalogos');
  }

  elegirCooperativa(id) {
    this.cooperativaID = id ? String(id) : '';
    const candidatas = this.cuentasDeCooperativa();
    if (!candidatas.some((c) => String(c.id) === String(this.cuentaID))) {
      this.cuentaID = candidatas.length ? String(candidatas[0].id) : '';
    }
    this.anunciar('contexto');
  }

  elegirCuenta(id) {
    this.cuentaID = id ? String(id) : '';
    const cuenta = this.cuenta;
    if (cuenta) this.cooperativaID = String(cuenta.cooperativa_id);
    this.anunciar('contexto');
  }

  anunciar(motivo) {
    this.dispatchEvent(new CustomEvent('cambio', { detail: { motivo } }));
  }
}

export const estado = new Estado();
