// Re-exporta do @lucide/vue mantendo os nomes existentes usados em todo o projeto.
// Tamanho padrão 20px aplicado via CSS (.nav-icon :deep(svg)) no AppSidebar.
export {
  LayoutGrid   as IconGrid,
  Building2    as IconBuilding,
  Factory      as IconFactory,
  Users        as IconUsers,
  KeyRound     as IconKey,
  RefreshCw    as IconSync,
  User         as IconUser,
  ChevronUp    as IconChevronUp,
  ChevronDown  as IconChevronDown,
  X            as IconX,
  Check        as IconCheck,
  Plus         as IconPlus,
  Pencil       as IconPencil,
  Trash2       as IconTrash,
  RefreshCw    as IconRefresh,
  Database     as IconDatabase,
  // Assistente de IA. Sparkles e o vocabulario que o mercado ja usa para
  // recurso de IA, entao o icone se explica sozinho no menu.
  Sparkles     as IconAssistente,
  // Cards de KPI da Visão Geral. Substituem os glifos ↑ ↓ ◈ ⬡ que eram texto:
  // como texto eles herdavam métrica de fonte e não alinhavam dentro do quadrado.
  ArrowUpRight   as IconArrowUpRight,
  ArrowDownRight as IconArrowDownRight,
  LineChart      as IconLineChart,
  CreditCard     as IconCreditCard,
} from '@lucide/vue'
