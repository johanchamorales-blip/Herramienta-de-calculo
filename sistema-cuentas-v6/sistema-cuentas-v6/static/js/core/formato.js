// Formato de moneda y fechas para Guatemala.
const quetzales = new Intl.NumberFormat('es-GT', {
  style: 'currency',
  currency: 'GTQ',
  minimumFractionDigits: 2,
});

export function dinero(valor) {
  return quetzales.format(Number(valor) || 0);
}

// Las fechas del servidor vienen como "2026-09-30" (día calendario).
export function fecha(valor) {
  if (!valor) return '';
  const [anio, mes, dia] = String(valor).slice(0, 10).split('-');
  if (!anio || !mes || !dia) return '';
  return `${dia}/${mes}/${anio}`;
}

export function fechaHora(valor) {
  if (!valor) return '';
  const d = new Date(valor);
  if (Number.isNaN(d.getTime())) return '';
  return d.toLocaleString('es-GT', { dateStyle: 'short', timeStyle: 'short' });
}

export function hoy() {
  const d = new Date();
  const mes = String(d.getMonth() + 1).padStart(2, '0');
  const dia = String(d.getDate()).padStart(2, '0');
  return `${d.getFullYear()}-${mes}-${dia}`;
}

export const MESES = [
  'Enero', 'Febrero', 'Marzo', 'Abril', 'Mayo', 'Junio',
  'Julio', 'Agosto', 'Septiembre', 'Octubre', 'Noviembre', 'Diciembre',
];

export function periodo(mes, anio) {
  if (!mes || !anio) return 'Histórico completo';
  return `${MESES[mes - 1]} ${anio}`;
}

export function nombreRol(rol) {
  return {
    ADMINISTRADOR: 'Administrador',
    CONTADOR: 'Contador',
    JEFE_DE_PLANTA: 'Jefe de planta',
  }[rol] || rol;
}
