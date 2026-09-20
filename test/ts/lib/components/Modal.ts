export interface ModalProps {
  title: string;
  isOpen: boolean;
}

export function renderModal(props: ModalProps): string {
  return props.isOpen ? `Modal(${props.title})` : "";
}
