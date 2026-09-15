import { api } from '../core/api.js';
import { datosFormulario, escapar } from '../core/dom.js';
import { problema, exito } from '../core/notificaciones.js';

// Pantalla de acceso. Cuando el sistema está vacío pide crear el primer
// administrador; después solo permite iniciar sesión.
export class Acceso {
  constructor(contenedor, alEntrar) {
    this.contenedor = contenedor;
    this.alEntrar = alEntrar;
    this.modo = 'login';
  }

  async mostrar(sesion) {
    this.requiereAlta = sesion.requiere_alta;
    this.modo = this.requiereAlta ? 'registro' : 'login';
    this.pintar();
  }

  pintar() {
    const primeraVez = this.requiereAlta;
    const esRegistro = this.modo === 'registro';

    this.contenedor.innerHTML = `
      <div class="acceso">
        <section class="acceso-relato">
          <h1>El libro de bancos de tus cooperativas, al día</h1>
          <p>Registra depósitos y cheques, sigue el saldo corrido y concilia contra el estado de cuenta sin sacar la calculadora.</p>
          <div class="muestra">
            <table>
              <thead><tr><th>Fecha</th><th>Cheque</th><th>Beneficiario</th><th class="cifra">Cheques</th><th class="cifra">Saldo</th></tr></thead>
              <tbody>
                <tr><td>05/09</td><td>DEP-001</td><td>Transferencia planilla</td><td class="cifra"></td><td class="cifra">Q 13,000.00</td></tr>
                <tr><td>10/09</td><td>000145231</td><td>Ministerio de Trabajo</td><td class="cifra cheque">Q 4,000.00</td><td class="cifra">Q 9,000.00</td></tr>
                <tr><td>10/09</td><td>000145232</td><td>Gasolinera La Torre</td><td class="cifra cheque">Q 800.00</td><td class="cifra">Q 8,200.00</td></tr>
              </tbody>
            </table>
          </div>
        </section>

        <section class="acceso-formulario">
          ${esRegistro ? this.formularioRegistro(primeraVez) : this.formularioLogin()}
        </section>
      </div>`;

    this.contenedor.querySelector('form').addEventListener('submit', (evento) => {
      evento.preventDefault();
      if (esRegistro) this.registrar(evento.target);
      else this.entrar(evento.target);
    });

    const cambio = this.contenedor.querySelector('[data-accion="cambiar"]');
    cambio?.addEventListener('click', () => {
      this.modo = this.modo === 'login' ? 'registro' : 'login';
      this.pintar();
    });
  }

  formularioLogin() {
    return `
      <form>
        <h2>Iniciar sesión</h2>
        <p class="tenue" style="font-size:13px">Usa tu usuario o tu correo.</p>
        <label>Usuario o correo<input name="usuario" required autocomplete="username"></label>
        <label>Contraseña<input name="password" type="password" required autocomplete="current-password"></label>
        <button type="submit">Entrar</button>
        <p class="acceso-cambio">¿No tienes acceso? Pídelo al administrador del sistema.</p>
      </form>`;
  }

  formularioRegistro(primeraVez) {
    return `
      <form>
        <h2>${primeraVez ? 'Crear el primer administrador' : 'Crear cuenta'}</h2>
        <p class="tenue" style="font-size:13px">
          ${primeraVez
            ? 'Este usuario podrá gestionar catálogos, movimientos y accesos.'
            : 'Solo un administrador puede crear accesos nuevos.'}
        </p>
        <label>Usuario<input name="usuario" required minlength="3" autocomplete="username"></label>
        <label>Correo<input name="email" type="email" required autocomplete="email"></label>
        <label>Contraseña<input name="password" type="password" required minlength="8" autocomplete="new-password"></label>
        <label>Rol
          <select name="rol" ${primeraVez ? 'disabled' : ''}>
            <option value="ADMINISTRADOR">Administrador</option>
            <option value="CONTADOR">Contador</option>
            <option value="JEFE_DE_PLANTA">Jefe de planta</option>
          </select>
        </label>
        <button type="submit">${primeraVez ? 'Crear administrador' : 'Crear cuenta'}</button>
        ${primeraVez ? '' : '<p class="acceso-cambio"><button type="button" class="enlace" data-accion="cambiar">Volver a iniciar sesión</button></p>'}
      </form>`;
  }

  async entrar(form) {
    try {
      const sesion = await api.crear('/api/auth/login', datosFormulario(form));
      await this.alEntrar(sesion);
    } catch (error) {
      problema(error.message);
    }
  }

  async registrar(form) {
    const datos = datosFormulario(form);
    if (!datos.rol) datos.rol = 'ADMINISTRADOR';
    try {
      await api.crear('/api/auth/registro', datos);
      exito('Usuario creado. Ya puedes iniciar sesión.');
      this.requiereAlta = false;
      this.modo = 'login';
      this.pintar();
    } catch (error) {
      problema(escapar(error.message));
    }
  }
}
