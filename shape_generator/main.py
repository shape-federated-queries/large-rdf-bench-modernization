import glob
from shexer.shaper import Shaper
from shexer.consts import NT

namespaces_dict = {"http://www.w3.org/1999/02/22-rdf-syntax-ns#": "rdf",
                   "http://example.org/": "ex",
                   "http://weso.es/shapes/": "",
                   "http://www.w3.org/2001/XMLSchema#": "xsd"
                   }
file_path_glob = "/home/bryanelliott/Documents/PhD/coding/shape-index-federated-query-planning/dataset-cleaning/out/*.nt"

kg = ""
max = 5
i = 0
for file_path in glob.glob(file_path_glob):
    with open(file_path, 'r') as file:
        kg += file.read()
    i+=1
    if i == max:
        break

raw_graph = """
<http://example.org/sarah> <http://www.w3.org/1999/02/22-rdf-syntax-ns#type> <http://example.org/Person> .
<http://example.org/sarah> <http://example.org/age> "30"^^<http://www.w3.org/2001/XMLSchema#int> .
<http://example.org/sarah> <http://example.org/name> "Sarah" .
<http://example.org/sarah> <http://example.org/gender> <http://example.org/Female> .
<http://example.org/sarah> <http://example.org/occupation> <http://example.org/Doctor> .
<http://example.org/sarah> <http://example.org/brother> <http://example.org/Jim> .

<http://example.org/jim> <http://www.w3.org/1999/02/22-rdf-syntax-ns#type> <http://example.org/Person> .
<http://example.org/jim> <http://example.org/age> "28"^^<http://www.w3.org/2001/XMLSchema#int> .
<http://example.org/jim> <http://example.org/name> "Jimbo".
<http://example.org/jim> <http://example.org/surname> "Mendes".
<http://example.org/jim> <http://example.org/gender> <http://example.org/Male> .

<http://example.org/Male> <http://www.w3.org/1999/02/22-rdf-syntax-ns#type> <http://example.org/Gender> .
<http://example.org/Male> <http://www.w3.org/2000/01/rdf-schema#label> "Male" .
<http://example.org/Female> <http://www.w3.org/1999/02/22-rdf-syntax-ns#type> <http://example.org/Gender> .
<http://example.org/Female> <http://www.w3.org/2000/01/rdf-schema#label> "Female" .
<http://example.org/Other> <http://www.w3.org/1999/02/22-rdf-syntax-ns#type> <http://example.org/Gender> .
<http://example.org/Other> <http://www.w3.org/2000/01/rdf-schema#label> "Other gender" .
"""



input_nt_file = "target_graph.nt"

"""
shaper = Shaper(shape_map_raw=shape_map_raw,
                url_endpoint="https://query.wikidata.org/sparql",
                instantiation_property="http://www.wikidata.org/prop/direct/P31")
"""

shaper = Shaper(all_classes_mode=True,
                namespaces_dict=namespaces_dict,
                url_endpoint="http://localhost:8888")
"""
shaper = Shaper(all_classes_mode=True,
                raw_graph=kg,
                input_format=NT,
                namespaces_dict=namespaces_dict,  # Default: no prefixes
                instantiation_property="http://www.w3.org/1999/02/22-rdf-syntax-ns#type")  # Default rdf:type
"""

output_file = "shaper_example.shex"

shaper.shex_graph(output_file=output_file,
                  acceptance_threshold=0)

print("Done!")


# What can we play with
#
# For the complexity of the shape
# depth_for_building_subgraph (default 1): use this param just in case you are working against a SPARQL endpoint. This integer indicates the max distance from any seed node to consider in order to track a subgraph from the endpoint. Please, remind that a high depth can cause a massive number of queries and have a high performance cost.
#
# keep_less_specific (default True): when it is set to True, for a group of constraints with the same property and object but different cardinality, the one with less specific cardinality ('+') will be preserved, and the rest of constraints used to provide info in comments. When it is set to False, the preserved constraint will be the one with an integer as cardinality and the highest rate of conformance with the instances of the class.
#
# disable_or_statements (default True): when set to False, sheXer tries to infer constraints with the operator oneOf (|) in case there are constraints with the same property but different object. By default, sheXer groups those constraint in a isngle one having the less general object possible. For instance, when the objects are different shapes, it merges the constraints a single one whose object is IRI.
#
# allow_opt_cardinality (default True). When all-compliant mode is active, if there is a constraint which does not conform with every isntance but its maximun cardinality for any instance is {1}, it uses the optional cardinality (?). When set to False, it uses Kleene closure instead.
#
# disable_opt_cardinality (dafault False). When set to True, it prevents any constraint to have a higher cardinality higher than one, even if every instance has that cardinality. For example, a constraint such as ex:alias xsd:string {3} will be changed to ex:alias xsd:string +.
#
# detect_minimal_iri (default False). When it is set to True, each shape will be associated with a regex pattern. That pattern expresses the initial part of the IRI that is common to every isntance used to extract a given shape. This pattern is only serialized when it is a "worthy" one (long enough, not just "http://", etc.).
